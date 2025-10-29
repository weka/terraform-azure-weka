package scale_up

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/weka/go-cloud-lib/functions_def"
	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"

	"weka-deployment/common"
	"weka-deployment/functions/azure_functions_def"
)

var (
	stateStorageName   = os.Getenv("STATE_STORAGE_NAME")
	stateContainerName = os.Getenv("STATE_CONTAINER_NAME")
	stateBlobName      = os.Getenv("STATE_BLOB_NAME")
	prefix             = os.Getenv("PREFIX")
	clusterName        = os.Getenv("CLUSTER_NAME")
	subscriptionId     = os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName  = os.Getenv("RESOURCE_GROUP_NAME")
	nfsContainerName   = os.Getenv("NFS_STATE_CONTAINER_NAME")
	nfsStateBlobName   = os.Getenv("NFS_STATE_BLOB_NAME")
	nfsScaleSetName    = os.Getenv("NFS_VMSS_NAME")
	vmssConfigStr      = os.Getenv("VMSS_CONFIG")
	// initial state of the cluster
	initialClusterSize, _ = strconv.Atoi(os.Getenv("INITIAL_CLUSTER_SIZE"))
	clusterizeTarget, _   = strconv.Atoi(os.Getenv("CLUSTERIZATION_TARGET"))
	initialNfsSize, _     = strconv.Atoi(os.Getenv("NFS_PROTOCOL_GATEWAYS_NUM"))
	clusterInitialState   = protocol.ClusterState{
		InitialSize:          initialClusterSize,
		DesiredSize:          initialClusterSize,
		ClusterizationTarget: clusterizeTarget,
	}
	nfsInitialState = protocol.ClusterState{
		InitialSize:          initialNfsSize,
		DesiredSize:          initialNfsSize,
		ClusterizationTarget: initialNfsSize,
	}
)

func getBackendCustomDataScript(ctx context.Context, userData string) (customData string, err error) {
	functionAppName := os.Getenv("FUNCTION_APP_NAME")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")
	diskSize, _ := strconv.Atoi(os.Getenv("DISK_SIZE"))
	nicsNum, _ := strconv.Atoi(os.Getenv("NICS_NUM"))
	subnet := os.Getenv("SUBNET")
	aptRepo := os.Getenv("APT_REPO_SERVER")

	logger := logging.LoggerFromCtx(ctx)

	functionAppKey, err := common.GetKeyVaultValue(ctx, keyVaultUri, "function-app-default-key")
	if err != nil {
		logger.Error().Err(err).Msg("cannot get function app key")
		return
	}

	baseFunctionUrl := fmt.Sprintf("https://%s.azurewebsites.net/api/", functionAppName)
	funcDef := azure_functions_def.NewFuncDef(baseFunctionUrl, functionAppKey)
	reportFunction := funcDef.GetFunctionCmdDefinition(functions_def.Report)
	deployFunction := funcDef.GetFunctionCmdDefinition(functions_def.Deploy)
	fetchFunction := funcDef.GetFunctionCmdDefinition(functions_def.Fetch)

	// Get maintenance monitor script and service files
	monitorScript, err := common.GetMaintenanceMonitorScript(fetchFunction)
	if err != nil {
		logger.Error().Err(err).Msg("cannot get maintenance monitor script")
		return
	}

	serviceUnit, err := common.GetMaintenanceMonitorService()
	if err != nil {
		logger.Error().Err(err).Msg("cannot get maintenance monitor service")
		return
	}

	customDataStr := getInitScript(userData, diskSize, nicsNum, subnet, aptRepo, reportFunction, deployFunction, clusterName, monitorScript, serviceUnit)
	// base64 encode the custom data
	customData = base64.StdEncoding.EncodeToString([]byte(customDataStr))
	return
}

func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)
	stateParams := common.BlobObjParams{
		StorageName:   stateStorageName,
		ContainerName: stateContainerName,
		BlobName:      stateBlobName,
	}

	state, err := common.ReadStateOrCreateNew(ctx, stateParams, clusterInitialState)
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
	vmssConfig, err := common.ReadVmssConfig(ctx, vmssConfigStr)
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
		vmssParams := common.ScaleSetParams{
			SubscriptionId:    subscriptionId,
			ResourceGroupName: resourceGroupName,
			ScaleSetName:      vmScaleSetName,
		}
		handleProgressingClusterization(ctx, &state, vmssParams, stateParams)
		logger.Info().Msg(msg)
		returnMsg = msg
	} else {
		currentConfig := common.GetVmssConfig(ctx, resourceGroupName, scaleSet)

		// 2. Update flow: compare current vmss config with expected vmss config and update if needed
		if vmssConfig.ConfigHash != currentConfig.ConfigHash {
			diff := common.VmssConfigsDiff(*currentConfig, vmssConfig)
			logger.Info().Msgf("vmss config diff: %s", diff)

			err := handleVmssUpdate(ctx, currentConfig, &vmssConfig, stateParams, state.DesiredSize)
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
	logger.Info().Msg(returnMsg)

	// handle NFS vmss
	if nfsScaleSetName != "" {
		message, err := handleNFSScaleUp(ctx)
		if err != nil {
			common.WriteErrorResponse(w, err)
			return
		}
		returnMsg = fmt.Sprintf("%s; %s", returnMsg, message)
	}

	common.WriteSuccessResponse(w, returnMsg)
}

func handleNFSScaleUp(ctx context.Context) (message string, err error) {
	logger := logging.LoggerFromCtx(ctx)

	nfsStateParams := common.BlobObjParams{
		StorageName:   stateStorageName,
		ContainerName: nfsContainerName,
		BlobName:      nfsStateBlobName,
	}
	nfsState, err := common.ReadStateOrCreateNew(ctx, nfsStateParams, nfsInitialState)
	if err != nil {
		logger.Error().Err(err).Msg("cannot read NFS state")
		return
	}

	if !nfsState.Clusterized {
		message = fmt.Sprintf("NFS not clusterized yet, initial size %d is set", nfsState.InitialSize)

		vmssParams := common.ScaleSetParams{
			SubscriptionId:    subscriptionId,
			ResourceGroupName: resourceGroupName,
			ScaleSetName:      nfsScaleSetName,
			Flexible:          true,
		}
		handleProgressingClusterization(ctx, &nfsState, vmssParams, nfsStateParams)
		logger.Info().Msg(message)
	}

	err = common.ScaleUp(ctx, subscriptionId, resourceGroupName, nfsScaleSetName, int64(nfsState.DesiredSize))
	if err != nil {
		err = fmt.Errorf("cannot scale up NFS vmss: %v", err)
		return
	}
	message = fmt.Sprintf("scaled up NFS vmss %s to size %d successfully", nfsScaleSetName, nfsState.DesiredSize)
	logger.Info().Msg(message)
	return
}

func createVmss(ctx context.Context, vmssConfig *common.VMSSConfig, vmssName string, vmssSize int) (err error) {
	logger := logging.LoggerFromCtx(ctx)
	vmssConfigHash := vmssConfig.ConfigHash

	customData, err := getBackendCustomDataScript(ctx, vmssConfig.UserData)
	if err != nil {
		logger.Error().Err(err).Msg("cannot get custom data script")
		return err
	}

	logger.Info().Msgf("creating new vmss %s of size %d", vmssName, vmssSize)
	_, err = common.CreateOrUpdateVmss(ctx, subscriptionId, resourceGroupName, vmssName, vmssConfigHash, *vmssConfig, vmssSize, customData)
	if err != nil {
		return err
	}
	logger.Info().Msgf("created vmss %s", vmssName)
	return nil
}

func handleVmssUpdate(ctx context.Context, currentConfig, newConfig *common.VMSSConfig, stateParams common.BlobObjParams, desiredSize int) (err error) {
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
		return common.AddClusterUpdate(ctx, stateParams, update)
	}

	customData, err := getBackendCustomDataScript(ctx, newConfig.UserData)
	if err != nil {
		logger.Error().Err(err).Msg("cannot get custom data script")
		return err
	}

	_, err = common.CreateOrUpdateVmss(ctx, subscriptionId, resourceGroupName, currentConfig.Name, newConfigHash, *newConfig, desiredSize, customData)
	if err != nil {
		logger.Error().Err(err).Msgf("cannot update vmss %s", currentConfig.Name)
		errStr := err.Error()
		update.Error = &errStr
	} else {
		logger.Info().Msgf("updated vmss %s to new config_hash %s", currentConfig.Name, newConfigHash)
	}

	return common.AddClusterUpdate(ctx, stateParams, update)
}

func handleProgressingClusterization(ctx context.Context, state *protocol.ClusterState, vmssParams common.ScaleSetParams, stateParams common.BlobObjParams) {
	logger := logging.LoggerFromCtx(ctx)

	vms, err := common.GetScaleSetVmsExpandedView(ctx, &vmssParams)
	if err != nil {
		msg := fmt.Sprintf("Failed getting vms list for vmss %s: %v", vmssParams.ScaleSetName, err)
		common.ReportMsg(ctx, "vmss", stateParams, "error", msg)
		return
	}
	toTerminate := common.GetUnhealthyInstancesToTerminate(ctx, vms)
	if len(toTerminate) > 0 {
		msg := fmt.Sprintf("Terminating unhealthy instances indexes: %v", toTerminate)
		common.ReportMsg(ctx, "vmss", stateParams, "debug", msg)
	}

	_, terminateErrors := common.TerminateScaleSetInstances(ctx, &vmssParams, toTerminate)
	if len(terminateErrors) > 0 {
		msg := fmt.Sprintf("errors during terminating unhealthy instances: %v", terminateErrors)
		logger.Info().Msgf(msg)
		common.ReportMsg(ctx, "vmss", stateParams, "error", msg)
	}
}
