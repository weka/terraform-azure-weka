package scale_up

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"weka-deployment/common"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
)

func itemInList(item string, list []string) bool {
	for _, listItem := range list {
		if item == listItem {
			return true
		}
	}
	return false
}

const RefreshVmssInstancesAddingStep = 10

type ScaleUpEnvParams struct {
	StateContainerName   string
	StateStorageName     string
	VmssStateStorageName string
	SubscriptionId       string
	ResourceGroupName    string
	KeyVaultUri          string
	VmssName             string
	RefreshVmssName      string
	DesiredSize          int
}

func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	vmssStateStorageName := os.Getenv("VMSS_STATE_STORAGE_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")

	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		logger.Error().Err(err).Msg("cannot read state")
		common.WriteErrorResponse(w, err)
		return
	}

	vmssState, err := common.ReadVmssState(ctx, vmssStateStorageName, stateContainerName)
	if err != nil {
		logger.Error().Err(err).Msg("cannot read vmss state")
		common.WriteErrorResponse(w, err)
		return
	}

	version := vmssState.VmssVersion
	vmssName := common.GetVmScaleSetName(prefix, clusterName, version)
	refreshVmssName := common.GetRefreshVmssName(vmssName, version)

	envParams := ScaleUpEnvParams{
		StateContainerName:   stateContainerName,
		StateStorageName:     stateStorageName,
		VmssStateStorageName: vmssStateStorageName,
		SubscriptionId:       os.Getenv("SUBSCRIPTION_ID"),
		ResourceGroupName:    os.Getenv("RESOURCE_GROUP_NAME"),
		KeyVaultUri:          os.Getenv("KEY_VAULT_URI"),
		VmssName:             vmssName,
		RefreshVmssName:      refreshVmssName,
		DesiredSize:          state.DesiredSize,
	}

	// initiale vmss creation if needed
	if !vmssState.VmssCreated {
		err := HandleVmssCreate(ctx, &envParams, &vmssState)
		if err != nil {
			logger.Error().Err(err).Msgf("cannot create vmss %s", envParams.VmssName)
			common.WriteErrorResponse(w, err)
			return
		}
		common.WriteSuccessResponse(w, fmt.Sprintf("created vmss %s successfully", envParams.VmssName))
		return
	}

	// after vmss creation we need to wait until vmss is clusterized
	if !state.Clusterized {
		handleProgressingClusterization(ctx, &state, envParams.SubscriptionId, envParams.ResourceGroupName, envParams.VmssName, envParams.StateContainerName, envParams.StateStorageName)
		common.WriteSuccessResponse(w, "Not clusterized yet, skipping...")
		return
	}

	// get expected vmss config
	vmssConfig, err := common.ReadVmssConfig(ctx, envParams.VmssStateStorageName, envParams.StateContainerName)
	if err != nil {
		logger.Error().Err(err).Msgf("cannot read vmss config")
		common.WriteErrorResponse(w, err)
		return
	}

	// handle vmss refresh if needed
	if vmssState.RefreshStatus != common.RefreshNone {
		err := HandleVmssRefresh(ctx, &envParams, &vmssConfig, &vmssState)
		if err != nil {
			common.WriteErrorResponse(w, err)
			return
		}
		common.WriteSuccessResponse(w, "vmss refresh initiated successfully")
		return
	}

	// compare current vmss config with expected vmss config and update if needed
	if diff := common.VmssConfigsDiff(vmssState.CurrentConfig, &vmssConfig); diff != "" {
		logger.Info().Msgf("vmss config changed, diff: %s", diff)

		err := HandleVmssUpdate(ctx, &envParams, &vmssConfig, &vmssState)
		if err != nil {
			common.WriteErrorResponse(w, err)
			return
		}
		common.WriteSuccessResponse(w, "vmss update handled successfully")
		return
	}

	// scale up vmss if needed
	err = common.ScaleUp(ctx, envParams.SubscriptionId, envParams.ResourceGroupName, envParams.VmssName, int64(state.DesiredSize))
	if err != nil {
		common.WriteErrorResponse(w, err)
		return
	}
	common.WriteSuccessResponse(w, fmt.Sprintf("updated size to %d successfully", state.DesiredSize))
}

func HandleVmssCreate(ctx context.Context, params *ScaleUpEnvParams, vmssState *common.VMSSState) error {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("creating vmss %s", params.VmssName)

	functionAppName := os.Getenv("FUNCTION_APP_NAME")

	vmssConfig, err := common.ReadVmssConfig(ctx, params.VmssStateStorageName, params.StateContainerName)
	if err != nil {
		return err
	}

	vmssId, err := common.CreateOrUpdateVmss(ctx, params.SubscriptionId, params.ResourceGroupName, params.VmssName, vmssConfig, params.DesiredSize)
	if err != nil {
		return err
	}

	err = common.AssignVmssContributorRoleToFunctionApp(ctx, params.SubscriptionId, params.ResourceGroupName, *vmssId, functionAppName)
	if err != nil {
		err = fmt.Errorf("cannot assign vmss 'contributor' role to function app: %w", err)
		return err
	}

	logger.Info().Msgf("created vmss %s, updating vmss state", params.VmssName)
	vmssState.VmssCreated = true
	vmssState.CurrentConfig = &vmssConfig
	err = common.WriteVmssState(ctx, params.VmssStateStorageName, params.StateContainerName, *vmssState)
	return err
}

func HandleVmssRefresh(ctx context.Context, params *ScaleUpEnvParams, vmssConfig *common.VMSSConfig, vmssState *common.VMSSState) error {
	if vmssState.RefreshStatus == common.RefreshNeeded {
		return initiateVmssRefresh(ctx, params, vmssConfig, vmssState)
	} else if vmssState.RefreshStatus == common.RefreshInProgress {
		return progressVmssRefresh(ctx, params, vmssConfig, vmssState)
	} else {
		return fmt.Errorf("invalid refresh status: %d", vmssState.RefreshStatus)
	}
}

func HandleVmssUpdate(ctx context.Context, params *ScaleUpEnvParams, vmssConfig *common.VMSSConfig, vmssState *common.VMSSState) error {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("updating vmss %s", params.VmssName)

	if vmssState.CurrentConfig.SKU != vmssConfig.SKU {
		msg := fmt.Sprintf("cannot change vmss SKU from %s to %s, need refresh", vmssState.CurrentConfig.SKU, vmssConfig.SKU)
		logger.Info().Msg(msg)
		setNeedRefreshVmssState(ctx, params, vmssState)
		return fmt.Errorf(msg)
	}

	_, err := common.CreateOrUpdateVmss(ctx, params.SubscriptionId, params.ResourceGroupName, params.VmssName, *vmssConfig, params.DesiredSize)
	if err != nil {
		if updErr, ok := err.(*azcore.ResponseError); ok && updErr.ErrorCode == "PropertyChangeNotAllowed" {
			setNeedRefreshVmssState(ctx, params, vmssState)
		}
		logger.Error().Err(err).Msgf("cannot update vmss %s", params.VmssName)
		return err
	}

	logger.Info().Msgf("updated vmss %s", params.VmssName)
	return nil
}

func setNeedRefreshVmssState(ctx context.Context, params *ScaleUpEnvParams, vmssState *common.VMSSState) error {
	logger := logging.LoggerFromCtx(ctx)

	logger.Info().Msgf("need to refresh vmss %s", params.VmssName)
	vmssState.RefreshStatus = common.RefreshNeeded

	err := common.WriteVmssState(ctx, params.VmssStateStorageName, params.StateContainerName, *vmssState)
	if err != nil {
		err = fmt.Errorf("cannot update vmss state: %w", err)
		logger.Error().Err(err).Msgf("cannot update vmss %s", params.VmssName)
	}
	return err
}

func initiateVmssRefresh(ctx context.Context, params *ScaleUpEnvParams, vmssConfig *common.VMSSConfig, vmssState *common.VMSSState) error {
	// Make sure that vmss current size is equal to "desired" number of weka instances
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("starting vmss refresh for %s", params.VmssName)

	newVmssName := params.RefreshVmssName
	newVmssSize := 0
	// if public ip address is assigned to vmss, domainNameLabel should differ (avoid VMScaleSetDnsRecordsInUse error)
	for i := range vmssConfig.PrimaryNIC.IPConfigurations {
		newDnsLabelName := fmt.Sprintf("%s-v%d", vmssConfig.PrimaryNIC.IPConfigurations[i].PublicIPAddress.DomainNameLabel, vmssState.VmssVersion+1)
		vmssConfig.PrimaryNIC.IPConfigurations[i].PublicIPAddress.DomainNameLabel = newDnsLabelName
	}

	// update hostname prefix
	vmssConfig.ComputerNamePrefix = fmt.Sprintf("%s-v%d", vmssConfig.ComputerNamePrefix, vmssState.VmssVersion+1)

	logger.Info().Msgf("creating new vmss %s of size %d", newVmssName, newVmssSize)

	_, err := common.CreateOrUpdateVmss(ctx, params.SubscriptionId, params.ResourceGroupName, newVmssName, *vmssConfig, newVmssSize)
	if err != nil {
		err = fmt.Errorf("cannot create new vmss: %w", err)
		logger.Error().Err(err).Msgf("cannot create 'refresh' vmss %s", params.VmssName)
		return err
	}

	logger.Info().Msgf("created new vmss %s, updating vmss state", params.VmssName)
	vmssState.RefreshStatus = common.RefreshInProgress
	vmssState.CurrentConfig = vmssConfig
	err = common.WriteVmssState(ctx, params.VmssStateStorageName, params.StateContainerName, *vmssState)
	return err
}

func progressVmssRefresh(ctx context.Context, params *ScaleUpEnvParams, vmssConfig *common.VMSSConfig, vmssState *common.VMSSState) error {
	// Terminology:
	// "Outdated" vmss -- vmss that was used before refresh
	// "Refresh" vmss -- vmss that was created during refresh
	// "desired" number of weka instances -- number of weka instances expected by the user (stored in state)
	//   note: "desired" number of weka instances should be the same as Outdated vmss size
	// "temporarily expected" number of weka instances = desired number of weka instances + Refresh vmss size
	//
	// Algorithm:
	// 1. check current size of Refresh vmss
	// 2. check total number of weka instances in the weka cluster (Outdated vmss size + Refresh vmss size)
	// 3. if number of weka instances is equal to "temporarily expected" number of weka instances, then:
	//   - scale down Outdated vmss using the following formula:
	//     desired number of weka instances - Refresh vmss size
	// 4. if Refresh vmss size is less than desired number of weka instances
	//  and Outdated vmss size == (desired number of weka instances - Refresh vmss size), then:
	//   - scale up Refresh vmss to size defined bu 'calculateRefreshVmssSize' function
	// 5. if Refresh vmss size is equal to desired number of weka instances, then:
	//   - scale down Outdated vmss to 0
	// 6. if Outdated vmss size is 0, then:
	//   - delete Outdated vmss
	//   - rename Refresh vmss to Outdated vmss' name
	//   - update vmss state
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("progressing vmss refresh for %s", params.VmssName)

	refreshVmssSize, err := common.GetScaleSetSize(ctx, params.SubscriptionId, params.ResourceGroupName, params.RefreshVmssName)
	if err != nil {
		err = fmt.Errorf("cannot get refresh vmss size: %w", err)
		logger.Error().Err(err).Send()
		return err
	}

	outdatedVmssSize, err := common.GetScaleSetSize(ctx, params.SubscriptionId, params.ResourceGroupName, params.VmssName)
	if err != nil {
		err = fmt.Errorf("cannot get outdated vmss size: %w", err)
		logger.Error().Err(err).Send()
		return err
	}

	if outdatedVmssSize == params.DesiredSize-refreshVmssSize && outdatedVmssSize != 0 {
		newSize := calculateRefreshVmssSize(refreshVmssSize, params.DesiredSize)
		logger.Info().Msgf("scaling up refresh vmss %s from %d to %d", params.RefreshVmssName, refreshVmssSize, newSize)
		err = common.ScaleUp(ctx, params.SubscriptionId, params.ResourceGroupName, params.RefreshVmssName, int64(newSize))
		if err != nil {
			err = fmt.Errorf("cannot scale up refresh vmss: %w", err)
			logger.Error().Err(err).Send()
			return err
		}
		logger.Info().Msgf("scaled up refresh vmss from %d to %d", refreshVmssSize, newSize)
		return nil
	}

	// wekaClusterStatus, err := status.GetClusterStatus(ctx, params.SubscriptionId, params.ResourceGroupName, params.VmssName, params.StateStorageName, params.StateContainerName, params.KeyVaultUri, params.RefreshVmssName)
	// if err != nil {
	// 	err = fmt.Errorf("cannot get weka cluster status: %w", err)
	// 	logger.Error().Err(err).Send()
	// 	return err
	// }

	if outdatedVmssSize == 0 {
		logger.Info().Msgf("deleting outdated vmss %s", params.VmssName)
		err = common.DeleteVmss(ctx, params.SubscriptionId, params.ResourceGroupName, params.VmssName)
		if err != nil {
			err = fmt.Errorf("cannot delete outdated vmss: %w", err)
			logger.Error().Err(err).Send()
			return err
		}

		logger.Info().Msgf("updating vmss state")
		vmssState.RefreshStatus = common.RefreshNone
		vmssState.VmssVersion = vmssState.VmssVersion + 1
		vmssState.CurrentConfig = vmssConfig
		err = common.WriteVmssState(ctx, params.VmssStateStorageName, params.StateContainerName, *vmssState)
		if err != nil {
			err = fmt.Errorf("cannot update vmss state: %w", err)
			logger.Error().Err(err).Send()
			return err
		}
		logger.Info().Msgf("refresh vmss finished successfully, new vmss version is %d", vmssState.VmssVersion)
		return nil
	}
	return nil
}

func calculateRefreshVmssSize(current, expected int) int {
	if expected <= current {
		return expected
	}
	if expected-current < RefreshVmssInstancesAddingStep {
		return expected
	}
	return current + RefreshVmssInstancesAddingStep
}

func handleProgressingClusterization(ctx context.Context, state *protocol.ClusterState, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName string) {
	logger := logging.LoggerFromCtx(ctx)

	vms, err := common.GetScaleSetVmsExpandedView(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		msg := fmt.Sprintf("Failed getting vms list for vmss %s: %v", vmScaleSetName, err)
		common.ReportMsg(ctx, "vmss", subscriptionId, resourceGroupName, stateContainerName, stateStorageName, "error", msg)
	}
	toTerminate := common.GetUnhealthyInstancesToTerminate(ctx, vms)
	if len(toTerminate) > 0 {
		msg := fmt.Sprintf("Terminating unhealthy instances indexes: %v", toTerminate)
		common.ReportMsg(ctx, "vmss", subscriptionId, resourceGroupName, stateContainerName, stateStorageName, "debug", msg)
	}

	var readyForClusterization []string
	var inProgress []string
	var unknown []string
	var stopped []string
	var allVms []string

	for _, instance := range state.Instances {
		readyForClusterization = append(readyForClusterization, strings.Split(instance, ":")[1])
	}

	for _, vm := range vms {
		if vm.Properties.InstanceView.ComputerName != nil {
			allVms = append(allVms, *vm.Properties.InstanceView.ComputerName)
			if itemInList(common.GetScaleSetVmId(*vm.ID), toTerminate) {
				stopped = append(stopped, *vm.Properties.InstanceView.ComputerName)
			}
		}
	}

	for vmName := range state.Progress {
		if !itemInList(vmName, readyForClusterization) && !itemInList(vmName, stopped) && itemInList(vmName, allVms) {
			inProgress = append(inProgress, vmName)
		}
	}

	for _, vmName := range allVms {
		if !itemInList(vmName, readyForClusterization) && !itemInList(vmName, stopped) && !itemInList(vmName, inProgress) {
			unknown = append(unknown, vmName)
		}
	}

	clusterizationInstance := state.Summary.ClusterizationInstance
	if len(state.Instances) == state.InitialSize {
		clusterizationInstance = strings.Split(state.Instances[len(state.Instances)-1], ":")[1]
	}

	summary := protocol.ClusterizationStatusSummary{
		ReadyForClusterization: len(state.Instances),
		Stopped:                len(toTerminate),
		Unknown:                unknown,
		InProgress:             len(inProgress),
		ClusterizationInstance: clusterizationInstance,
	}

	_ = common.UpdateSummaryAndInProgress(ctx, stateContainerName, stateStorageName, summary, inProgress)

	_, terminateErrors := common.TerminateScaleSetInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, toTerminate)
	if len(terminateErrors) > 0 {
		msg := fmt.Sprintf("errors during terminating unhealthy instances: %v", terminateErrors)
		logger.Info().Msgf(msg)
		common.ReportMsg(ctx, "vmss", subscriptionId, resourceGroupName, stateContainerName, stateStorageName, "error", msg)
	}
}
