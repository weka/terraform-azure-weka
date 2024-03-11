package scale_up

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
)

var (
	stateStorageName   = os.Getenv("STATE_STORAGE_NAME")
	stateContainerName = os.Getenv("STATE_CONTAINER_NAME")
	prefix             = os.Getenv("PREFIX")
	clusterName        = os.Getenv("CLUSTER_NAME")
	subscriptionId     = os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName  = os.Getenv("RESOURCE_GROUP_NAME")
)

func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		logger.Error().Err(err).Msg("cannot read state")
		common.WriteErrorResponse(w, err)
		return
	}

	scaleSet, err := common.GetScaleSetOrNil(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		logger.Error().Err(err).Msg("cannot get scale set")
		common.WriteErrorResponse(w, err)
		return
	}

	if scaleSet == nil && (state.Clusterized || len(state.Instances) > 0) {
		err := fmt.Errorf("vmss %s is not found but state already contains clusterization info", vmScaleSetName)
		logger.Error().Err(err).Send()
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

	// 1. Initial VMSS creation flow: initiale vmss creation if needed
	if scaleSet == nil && !state.Clusterized && len(state.Instances) == 0 {
		err := createVmss(ctx, &vmssConfig, vmScaleSetName, state.InitialSize)
		if err != nil {
			logger.Error().Err(err).Msgf("cannot create initial vmss")
			common.WriteErrorResponse(w, err)
		} else {
			common.WriteSuccessResponse(w, "created initial vmss successfully")
		}
		return
	}

	returnMsg := ""
	// after vmss creation we need to wait until vmss is clusterized
	if !state.Clusterized {
		msg := fmt.Sprintf("Not clusterized yet, initial size %d is set", state.InitialSize)
		handleProgressingClusterization(ctx, &state, subscriptionId, resourceGroupName, *scaleSet.Name, stateContainerName, stateStorageName)
		logger.Info().Msg(msg)
		returnMsg = msg
	} else {
		currentConfig := common.GetVmssConfig(ctx, resourceGroupName, scaleSet)

		// 2. Update flow: compare current vmss config with expected vmss config and update if needed
		if vmssConfig.ConfigHash != currentConfig.ConfigHash {
			diff := common.VmssConfigsDiff(*currentConfig, vmssConfig)
			logger.Info().Msgf("vmss config diff: %s", diff)

			err := handleVmssUpdate(ctx, currentConfig, &vmssConfig, state.DesiredSize)
			if err != nil {
				common.WriteErrorResponse(w, err)
				return
			}
			common.WriteSuccessResponse(w, "vmss update handled successfully")
			return
		}
		returnMsg = "vmss is up to date"
	}

	// Scale up latest vmss if needed
	err = common.ScaleUp(ctx, subscriptionId, resourceGroupName, *scaleSet.Name, int64(state.DesiredSize))
	if err != nil {
		common.WriteErrorResponse(w, err)
		return
	}
	returnMsg = fmt.Sprintf("%s; scaled up vmss %s to size %d successfully", returnMsg, *scaleSet.Name, state.DesiredSize)
	common.WriteSuccessResponse(w, returnMsg)
}

func createVmss(ctx context.Context, vmssConfig *common.VMSSConfig, vmssName string, vmssSize int) error {
	logger := logging.LoggerFromCtx(ctx)
	vmssConfigHash := vmssConfig.ConfigHash

	logger.Info().Msgf("creating new vmss %s of size %d", vmssName, vmssSize)
	_, err := common.CreateOrUpdateVmss(ctx, subscriptionId, resourceGroupName, vmssName, vmssConfigHash, *vmssConfig, vmssSize)
	if err != nil {
		return err
	}
	logger.Info().Msgf("created vmss %s", vmssName)
	return nil
}

func handleVmssUpdate(ctx context.Context, currentConfig, newConfig *common.VMSSConfig, desiredSize int) error {
	logger := logging.LoggerFromCtx(ctx)

	newConfigHash := newConfig.ConfigHash
	logger.Info().Msgf("updating vmss %s (hash %s) to new config_hash %s", currentConfig.Name, currentConfig.ConfigHash, newConfigHash)

	update := protocol.Update{
		From: currentConfig.ConfigHash,
		To:   newConfigHash,
		Time: time.Now(),
	}

	if currentConfig.SKU != newConfig.SKU {
		err := fmt.Errorf("cannot update vmss %s SKU from %s to %s", currentConfig.Name, currentConfig.SKU, newConfig.SKU)
		logger.Error().Err(err).Send()
		errStr := err.Error()
		update.Error = &errStr
		return common.AddClusterUpdate(ctx, stateContainerName, stateStorageName, update)
	}

	_, err := common.CreateOrUpdateVmss(ctx, subscriptionId, resourceGroupName, currentConfig.Name, newConfigHash, *newConfig, desiredSize)
	if err != nil {
		logger.Error().Err(err).Msgf("cannot update vmss %s", currentConfig.Name)
		errStr := err.Error()
		update.Error = &errStr
	} else {
		logger.Info().Msgf("updated vmss %s to new config_hash %s", currentConfig.Name, newConfigHash)
	}

	return common.AddClusterUpdate(ctx, stateContainerName, stateStorageName, update)
}

func handleProgressingClusterization(ctx context.Context, state *protocol.ClusterState, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName string) {
	logger := logging.LoggerFromCtx(ctx)

	vms, err := common.GetScaleSetVmsExpandedView(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		msg := fmt.Sprintf("Failed getting vms list for vmss %s: %v", vmScaleSetName, err)
		common.ReportMsg(ctx, "vmss", stateContainerName, stateStorageName, "error", msg)
		return
	}
	toTerminate := common.GetUnhealthyInstancesToTerminate(ctx, vms)
	if len(toTerminate) > 0 {
		msg := fmt.Sprintf("Terminating unhealthy instances indexes: %v", toTerminate)
		common.ReportMsg(ctx, "vmss", stateContainerName, stateStorageName, "debug", msg)
	}

	_, terminateErrors := common.TerminateScaleSetInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, toTerminate)
	if len(terminateErrors) > 0 {
		msg := fmt.Sprintf("errors during terminating unhealthy instances: %v", terminateErrors)
		logger.Info().Msgf(msg)
		common.ReportMsg(ctx, "vmss", stateContainerName, stateStorageName, "error", msg)
	}
}
