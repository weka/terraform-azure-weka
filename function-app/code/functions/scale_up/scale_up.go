package scale_up

import (
	"context"
	"fmt"
	"net/http"
	"os"
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

	vmssState, err := common.ReadVmssState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		err = fmt.Errorf("cannot read vmss state: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	scaleSetNames := common.GetScaleSetsNamesFromVmssState(ctx, subscriptionId, resourceGroupName, &vmssState)
	scaleSets, err := common.GetScaleSetsList(ctx, subscriptionId, resourceGroupName, scaleSetNames)
	if err != nil {
		logger.Error().Err(err).Msgf("cannot get scale sets list")
		common.WriteErrorResponse(w, err)
		return
	}

	// get expected vmss config
	vmssConfig, err := common.ReadVmssConfig(ctx, stateStorageName, stateContainerName)
	if err != nil {
		logger.Error().Err(err).Msgf("cannot read vmss config")
		common.WriteErrorResponse(w, err)
		return
	}

	targetVersion, err := common.GetConfigHash(vmssConfig)
	if err != nil {
		logger.Error().Err(err).Msgf("cannot get vmss config hash")
		common.WriteErrorResponse(w, err)
		return
	}

	// 1. Initial VMSS creation flow: initiale vmss creation if needed
	if vmssState.IsEmpty() && !state.Clusterized {
		err := createVmss(ctx, &vmssConfig, &vmssState, state.DesiredSize, targetVersion)
		if err != nil {
			logger.Error().Err(err).Msgf("cannot create initial vmss")
			common.WriteErrorResponse(w, err)
		} else {
			common.WriteSuccessResponse(w, "created initial vmss successfully")
		}
		return
	}

	if len(scaleSets) == 0 && !vmssState.IsEmpty() {
		err := fmt.Errorf("cannot find scale sets %v", scaleSetNames)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	latestVmss := scaleSets[len(scaleSets)-1]
	currentConfig := common.GetVmssConfig(ctx, resourceGroupName, latestVmss)

	// after vmss creation we need to wait until vmss is clusterized
	if !state.Clusterized {
		handleProgressingClusterization(ctx, &state, subscriptionId, resourceGroupName, *latestVmss.Name, stateContainerName, stateStorageName)
		logger.Info().Msgf("cluster is not clusterized yet, skipping...")
		common.WriteSuccessResponse(w, "Not clusterized yet, skipping...")
		return
	}

	// 2. Update flow: compare current vmss config with expected vmss config and update if needed
	if !targetVersionIsLatestVersion(vmssState.Versions, targetVersion) {
		diff := common.VmssConfigsDiff(*currentConfig, vmssConfig)
		logger.Info().Msgf("vmss config diff: %s", diff)

		err := HandleVmssUpdate(ctx, currentConfig, &vmssConfig, &vmssState, state.DesiredSize, targetVersion)
		if err != nil {
			common.WriteErrorResponse(w, err)
			return
		}
		common.WriteSuccessResponse(w, "vmss update handled successfully")
		return
	}

	returnMsg := ""
	// 3. Refresh in progress flow: handle vmss refresh if needed
	if len(scaleSets) > 1 && targetVersionIsLatestVersion(vmssState.Versions, targetVersion) {
		err := progressVmssRefresh(ctx, scaleSets, &vmssState, state.DesiredSize)
		if err != nil {
			common.WriteErrorResponse(w, err)
			return
		}
		returnMsg = "progressed vmss refresh successfully"
	}

	// Scale up latest vmss if needed
	err = common.ScaleUp(ctx, subscriptionId, resourceGroupName, *latestVmss.Name, int64(state.DesiredSize))
	if err != nil {
		common.WriteErrorResponse(w, err)
		return
	}
	returnMsg = fmt.Sprintf("%s; scaled up vmss %s to size %d successfully", returnMsg, *latestVmss.Name, state.DesiredSize)
	common.WriteSuccessResponse(w, returnMsg)
}

func HandleVmssUpdate(ctx context.Context, currentConfig, newConfig *common.VMSSConfig, vmssState *common.VMSSState, desiredSize int, newVersion string) error {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("updating vmss %s", currentConfig.Name)

	// leaseId, err := common.LockContainer(ctx, stateStorageName, stateContainerName)
	// if err != nil {
	// 	return err
	// }
	// defer common.UnlockContainer(ctx, stateStorageName, stateContainerName, leaseId)

	refreshNeeded := false
	if currentConfig.SKU != newConfig.SKU {
		msg := fmt.Sprintf("cannot change vmss SKU from %s to %s, need refresh", currentConfig.SKU, newConfig.SKU)
		logger.Info().Msg(msg)
		refreshNeeded = true
	} else {
		_, err := common.CreateOrUpdateVmss(ctx, subscriptionId, resourceGroupName, currentConfig.Name, newVersion, *newConfig, desiredSize)
		if err != nil {
			if updErr, ok := err.(*azcore.ResponseError); ok && updErr.ErrorCode == "PropertyChangeNotAllowed" {
				refreshNeeded = true
			}
			logger.Error().Err(err).Msgf("cannot update vmss %s", currentConfig.Name)
			return err
		} else {
			// replace active version in vmss state
			vmssState.ReplaceVersion(currentConfig.Tags["version"], newVersion)
			err = common.WriteVmssState(ctx, stateStorageName, stateContainerName, *vmssState)
			if err != nil {
				err = fmt.Errorf("cannot write vmss state: %w", err)
				return err
			}
		}
	}

	if refreshNeeded {
		err := initiateVmssRefresh(ctx, newConfig, vmssState, newVersion, desiredSize)
		if err != nil {
			logger.Error().Err(err).Msgf("cannot initiate vmss refresh")
			return err
		}
		logger.Info().Msgf("initiated vmss refresh")
		return nil
	}

	logger.Info().Msgf("updated vmss %s", currentConfig.Name)
	return nil
}

func initiateVmssRefresh(ctx context.Context, vmssConfig *common.VMSSConfig, vmssState *common.VMSSState, newVersion string, desiredSize int) error {
	// Make sure that vmss current size is equal to "desired" number of weka instances
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("initiate vmss refresh")

	newVmssName := common.GetVmScaleSetName(prefix, clusterName, newVersion)

	// if public ip address is assigned to vmss, domainNameLabel should differ (avoid VMScaleSetDnsRecordsInUse error)
	for i := range vmssConfig.PrimaryNIC.IPConfigurations {
		newDnsLabelName := fmt.Sprintf("%s-v%s", vmssConfig.PrimaryNIC.IPConfigurations[i].PublicIPAddress.DomainNameLabel, newVersion)
		vmssConfig.PrimaryNIC.IPConfigurations[i].PublicIPAddress.DomainNameLabel = newDnsLabelName
	}

	// update hostname prefix
	vmssConfig.ComputerNamePrefix = fmt.Sprintf("%s-v%s", vmssConfig.ComputerNamePrefix, newVersion)

	logger.Info().Msgf("creating new vmss %s of size %d", newVmssName, desiredSize)

	err := createVmss(ctx, vmssConfig, vmssState, desiredSize, newVersion)
	if err != nil {
		logger.Error().Err(err).Msgf("cannot create 'refresh' vmss %s", newVmssName)
		return err
	}
	return nil
}

func progressVmssRefresh(ctx context.Context, activeScaleSets []*armcompute.VirtualMachineScaleSet, vmssState *common.VMSSState, desiredSize int) error {
	// Terminology:
	// "Outdated" vmss -- vmss that was used before refresh
	// "Refresh" vmss -- vmss that was created during refresh
	// "desired" number of weka instances -- number of weka instances expected by the user (stored in state)
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("progressing vmss refresh")
	logger.Info().Msgf("active scale sets number: %d", len(activeScaleSets))

	latestVmss := activeScaleSets[len(activeScaleSets)-1]
	latestVmssSize := int(*latestVmss.SKU.Capacity)
	outdatedVmssTotalSize := 0

	for _, vmss := range activeScaleSets[:len(activeScaleSets)-1] {
		size := int(*vmss.SKU.Capacity)
		outdatedVmssTotalSize += size

		if size == 0 {
			logger.Info().Msgf("deleting outdated vmss %s with size %d", *vmss.Name, size)
			err := common.DeleteVmss(ctx, subscriptionId, resourceGroupName, *vmss.Name)
			if err != nil {
				err = fmt.Errorf("cannot delete outdated vmss: %w", err)
				logger.Error().Err(err).Send()
				return err
			}
			// delete outdated hash from vmss state
			version := vmss.Tags["version"]
			err = vmssState.RemoveVersion(*version)
			if err != nil {
				logger.Error().Err(err).Send()
				return err
			}
			err = common.WriteVmssState(ctx, stateStorageName, stateContainerName, *vmssState)
			if err != nil {
				logger.Error().Msgf("cannot write vmss state: %v", err)
				return err
			}
		}
	}

	logger.Info().Msgf("refresh vmss (%s) size is %d, outdated vmss(es) total size is %d", *latestVmss.Name, latestVmssSize, outdatedVmssTotalSize)
	return nil
}

func createVmss(ctx context.Context, vmssConfig *common.VMSSConfig, vmssState *common.VMSSState, vmssSize int, vmssVersion string) error {
	logger := logging.LoggerFromCtx(ctx)

	vmssName := common.GetVmScaleSetName(prefix, clusterName, vmssVersion)
	logger.Info().Msgf("creating vmss %s", vmssName)

	vmssId, err := common.CreateOrUpdateVmss(ctx, subscriptionId, resourceGroupName, vmssName, vmssVersion, *vmssConfig, vmssSize)
	if err != nil {
		return err
	}

	vmssState.AddVersion(vmssVersion)
	err = common.WriteVmssState(ctx, stateStorageName, stateContainerName, *vmssState)
	if err != nil {
		err = fmt.Errorf("cannot write vmss state: %w", err)
		return err
	}

	err = common.AssignVmssContributorRoleToFunctionApp(ctx, subscriptionId, resourceGroupName, *vmssId, functionAppName)
	if err != nil {
		if getErr, ok := err.(*azcore.ResponseError); ok && getErr.ErrorCode == "RoleAssignmentExists" {
			logger.Info().Msgf("vmss %s 'contributor' role is already assigned to function app", vmssName)
			return nil
		}
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

func targetVersionIsLatestVersion(activeVersions []string, targetVersion string) bool {
	if len(activeVersions) == 0 {
		return false
	}
	return activeVersions[len(activeVersions)-1] == targetVersion
}
