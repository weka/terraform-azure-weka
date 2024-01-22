package scale_up

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"weka-deployment/common"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"

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

const instancesAddingStep = 6

var (
	functionAppName    = os.Getenv("FUNCTION_APP_NAME")
	stateStorageName   = os.Getenv("STATE_STORAGE_NAME")
	stateContainerName = os.Getenv("STATE_CONTAINER_NAME")
	prefix             = os.Getenv("PREFIX")
	clusterName        = os.Getenv("CLUSTER_NAME")
	subscriptionId     = os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName  = os.Getenv("RESOURCE_GROUP_NAME")
)

type ScaleUpParams struct {
	VmssName        string
	RefreshVmssName string
	DesiredSize     int
}

func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		logger.Error().Err(err).Msg("cannot read state")
		common.WriteErrorResponse(w, err)
		return
	}

	scaleSets, err := common.GetScaleSetsOrderdedByVersion(ctx, subscriptionId, resourceGroupName, clusterName)
	if err != nil {
		logger.Error().Err(err).Msg("cannot get scale sets")
		common.WriteErrorResponse(w, err)
		return
	}
	logger.Info().Msgf("scale sets number: %d", len(scaleSets))

	// 1. Refresh in progress flow: handle vmss refresh if needed
	if len(scaleSets) > 1 {
		initialVmss, refreshVmss := scaleSets[0], scaleSets[1]
		err := progressVmssRefresh(ctx, initialVmss, refreshVmss, state.DesiredSize)
		if err != nil {
			common.WriteErrorResponse(w, err)
			return
		}
		common.WriteSuccessResponse(w, "progress vmss refresh succeeded")
		return
	}

	// get expected vmss config
	vmssConfig, err := common.ReadVmssConfig(ctx, stateStorageName, stateContainerName)
	if err != nil {
		logger.Error().Err(err).Msgf("cannot read vmss config")
		common.WriteErrorResponse(w, err)
		return
	}

	// 2. Initial VMSS creation flow: initiale vmss creation if needed
	if len(scaleSets) == 0 && !state.Clusterized {
		vmssVersion := 0
		err := createVmss(ctx, &vmssConfig, state.DesiredSize, vmssVersion)
		if err != nil {
			logger.Error().Err(err).Msgf("cannot create initial vmss")
			common.WriteErrorResponse(w, err)
		} else {
			common.WriteSuccessResponse(w, "created initial vmss successfully")
		}
		return
	} else if len(scaleSets) == 0 && state.Clusterized {
		msg := "need to remove state file and re-apply terraform"
		logger.Info().Msg(msg)
		common.WriteSuccessResponse(w, msg)
		return
	}

	initialVmss := scaleSets[0]

	// after vmss creation we need to wait until vmss is clusterized
	if !state.Clusterized {
		handleProgressingClusterization(ctx, &state, subscriptionId, resourceGroupName, *initialVmss.Name, stateContainerName, stateStorageName)
		logger.Info().Msgf("cluster is not clusterized yet, skipping...")
		common.WriteSuccessResponse(w, "Not clusterized yet, skipping...")
		return
	}

	currentConfig := common.GetVmssConfig(ctx, resourceGroupName, initialVmss)

	// 3. Update flow: compare current vmss config with expected vmss config and update if needed
	if diff := common.VmssConfigsDiff(*currentConfig, vmssConfig); diff != "" {
		logger.Info().Msgf("vmss config changed, diff: %s", diff)

		err := HandleVmssUpdate(ctx, initialVmss, currentConfig, &vmssConfig, state.DesiredSize)
		if err != nil {
			common.WriteErrorResponse(w, err)
			return
		}
		common.WriteSuccessResponse(w, "vmss update handled successfully")
		return
	}

	// 4. Scale up vmss if needed
	err = common.ScaleUp(ctx, subscriptionId, resourceGroupName, *initialVmss.Name, int64(state.DesiredSize))
	if err != nil {
		common.WriteErrorResponse(w, err)
		return
	}
	common.WriteSuccessResponse(w, fmt.Sprintf("updated size to %d successfully", state.DesiredSize))
}

func HandleVmssUpdate(ctx context.Context, initialVmss *armcompute.VirtualMachineScaleSet, currentConfig, newConfig *common.VMSSConfig, desiredSize int) error {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("updating vmss %s", *initialVmss.Name)

	currentVersionStr := initialVmss.Tags["version"]
	currentVersion, _ := strconv.Atoi(*currentVersionStr)
	newVersion := currentVersion + 1

	refreshNeeded := false
	if currentConfig.SKU != newConfig.SKU {
		msg := fmt.Sprintf("cannot change vmss SKU from %s to %s, need refresh", currentConfig.SKU, newConfig.SKU)
		logger.Info().Msg(msg)
		refreshNeeded = true
	} else {
		_, err := common.CreateOrUpdateVmss(ctx, subscriptionId, resourceGroupName, *initialVmss.Name, *newConfig, desiredSize, currentVersion)
		if err != nil {
			if updErr, ok := err.(*azcore.ResponseError); ok && updErr.ErrorCode == "PropertyChangeNotAllowed" {
				refreshNeeded = true
			}
			logger.Error().Err(err).Msgf("cannot update vmss %s", *initialVmss.Name)
			return err
		}
	}

	if refreshNeeded {
		err := initiateVmssRefresh(ctx, newConfig, newVersion)
		if err != nil {
			logger.Error().Err(err).Msgf("cannot initiate vmss refresh")
			return err
		}
		logger.Info().Msgf("initiated vmss refresh")
		return nil
	}

	logger.Info().Msgf("updated vmss %s", *initialVmss.Name)
	return nil
}

func initiateVmssRefresh(ctx context.Context, vmssConfig *common.VMSSConfig, newVersion int) error {
	// Make sure that vmss current size is equal to "desired" number of weka instances
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("initiate vmss refresh")

	newVmssName := common.GetVmScaleSetName(prefix, clusterName, newVersion)
	newVmssSize := 0

	// if public ip address is assigned to vmss, domainNameLabel should differ (avoid VMScaleSetDnsRecordsInUse error)
	for i := range vmssConfig.PrimaryNIC.IPConfigurations {
		newDnsLabelName := fmt.Sprintf("%s-v%d", vmssConfig.PrimaryNIC.IPConfigurations[i].PublicIPAddress.DomainNameLabel, newVersion)
		vmssConfig.PrimaryNIC.IPConfigurations[i].PublicIPAddress.DomainNameLabel = newDnsLabelName
	}

	// update hostname prefix
	vmssConfig.ComputerNamePrefix = fmt.Sprintf("%s-v%d", vmssConfig.ComputerNamePrefix, newVersion)

	logger.Info().Msgf("creating new vmss %s of size %d", newVmssName, newVmssSize)

	err := createVmss(ctx, vmssConfig, newVmssSize, newVersion)
	if err != nil {
		logger.Error().Err(err).Msgf("cannot create 'refresh' vmss %s", newVmssName)
		return err
	}
	return nil
}

func progressVmssRefresh(ctx context.Context, outdatedVmss, refreshVmss *armcompute.VirtualMachineScaleSet, desiredSize int) error {
	// Terminology:
	// "Outdated" vmss -- vmss that was used before refresh
	// "Refresh" vmss -- vmss that was created during refresh
	// "desired" number of weka instances -- number of weka instances expected by the user (stored in state)
	//   note: "desired" number of weka instances should be the same as Outdated vmss size
	//
	// Algorithm:
	// 1. check current size of Refresh vmss
	// 2. check total number of weka instances in the weka cluster (Outdated vmss size + Refresh vmss size)
	// 3. when instances of refresh vmss joined weka cluster, then
	//   - the old instances in Outdated vmss will be removed automatically by scale_down workflow
	// 4. if Refresh vmss size is less than desired number of weka instances
	//  and Outdated vmss size == (desired number of weka instances - Refresh vmss size), then:
	//   - scale up Refresh vmss to size defined bu 'calculateRefreshVmssSize' function
	// 5. if Refresh vmss size is equal to desired number of weka instances, then:
	//   - scale down Outdated vmss to 0
	// 6. if Outdated vmss size is 0, then:
	//   - delete Outdated vmss
	//   - set new vmss version
	//   - update vmss state
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("progressing vmss refresh for %s", *outdatedVmss.Name)

	refreshVmssSize := int(*refreshVmss.SKU.Capacity)
	outdatedVmssSize := int(*outdatedVmss.SKU.Capacity)

	logger.Info().Msgf("refresh vmss size is %d, outdated vmss size is %d", refreshVmssSize, outdatedVmssSize)

	if outdatedVmssSize == desiredSize-refreshVmssSize && outdatedVmssSize != 0 {
		newSize := calculateNewVmssSize(refreshVmssSize, desiredSize)
		logger.Info().Msgf("scaling up refresh vmss %s from %d to %d", *refreshVmss.Name, refreshVmssSize, newSize)
		err := common.ScaleUp(ctx, subscriptionId, resourceGroupName, *refreshVmss.Name, int64(newSize))
		if err != nil {
			err = fmt.Errorf("cannot scale up refresh vmss: %w", err)
			logger.Error().Err(err).Send()
			return err
		}
		logger.Info().Msgf("scaled up refresh vmss from %d to %d", refreshVmssSize, newSize)
		return nil
	}

	if outdatedVmssSize == 0 {
		logger.Info().Msgf("deleting outdated vmss %s", *outdatedVmss.Name)
		err := common.DeleteVmss(ctx, subscriptionId, resourceGroupName, *outdatedVmss.Name)
		if err != nil {
			err = fmt.Errorf("cannot delete outdated vmss: %w", err)
			logger.Error().Err(err).Send()
			return err
		}
	}
	return nil
}

func calculateNewVmssSize(current, expected int) int {
	if expected <= current {
		return expected
	}
	if expected-current < instancesAddingStep {
		return expected
	}
	return current + instancesAddingStep
}

func createVmss(ctx context.Context, vmssConfig *common.VMSSConfig, vmssSize, vmssVersion int) error {
	logger := logging.LoggerFromCtx(ctx)

	vmssName := common.GetVmScaleSetName(prefix, clusterName, vmssVersion)
	logger.Info().Msgf("creating vmss %s", vmssName)

	vmssId, err := common.CreateOrUpdateVmss(ctx, subscriptionId, resourceGroupName, vmssName, *vmssConfig, vmssSize, vmssVersion)
	if err != nil {
		return err
	}

	err = common.AssignVmssContributorRoleToFunctionApp(ctx, subscriptionId, resourceGroupName, *vmssId, functionAppName)
	if err != nil {
		err = fmt.Errorf("cannot assign vmss 'contributor' role to function app: %w", err)
		return err
	}

	logger.Info().Msgf("created vmss %s", vmssName)
	return nil
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
