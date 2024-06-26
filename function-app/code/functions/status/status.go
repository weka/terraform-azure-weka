package status

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/connectors"
	"github.com/weka/go-cloud-lib/lib/jrpc"
	"github.com/weka/go-cloud-lib/lib/weka"
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

func addSummary(ctx context.Context, state protocol.ClusterState, stateParams common.BlobObjParams, vmssParams *common.ScaleSetParams, reports *protocol.Reports) {
	logger := logging.LoggerFromCtx(ctx)

	if state.Clusterized {
		summary := protocol.ClusterizationStatusSummary{
			ClusterizationTarget: state.ClusterizationTarget,
			Clusterized:          state.Clusterized,
		}
		reports.Summary = summary
		return
	}

	vms, err := common.GetScaleSetVmsExpandedView(ctx, vmssParams)
	if err != nil {
		msg := fmt.Sprintf("Failed getting vms list for vmss %s: %v", vmssParams.ScaleSetName, err)
		logger.Error().Msg(msg)
		return
	}
	toTerminate := common.GetUnhealthyInstancesToTerminate(ctx, vms)

	var readyForClusterization []string
	var inProgress []string
	var unknown []string
	var stopped []string
	var allVms []string

	for _, instance := range state.Instances {
		readyForClusterization = append(readyForClusterization, strings.Split(instance.Name, ":")[1])
	}

	for _, vm := range vms {
		if vm.ComputerName != nil {
			allVms = append(allVms, *vm.ComputerName)
			if itemInList(common.GetScaleSetVmId(vm.ID), toTerminate) {
				stopped = append(stopped, *vm.ComputerName)
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

	clusterizationInstance := ""
	if len(state.Instances) >= state.ClusterizationTarget {
		clusterizationInstance = strings.Split(state.Instances[state.ClusterizationTarget-1].Name, ":")[1]
	}

	summary := protocol.ClusterizationStatusSummary{
		ReadyForClusterization: len(state.Instances),
		Stopped:                len(toTerminate),
		Unknown:                unknown,
		InProgress:             len(inProgress),
		ClusterizationInstance: clusterizationInstance,
		ClusterizationTarget:   state.ClusterizationTarget,
		Clusterized:            state.Clusterized,
	}

	reports.InProgress = inProgress
	reports.Summary = summary
}

func GetReports(ctx context.Context, stateParams common.BlobObjParams, vmssParams *common.ScaleSetParams) (reports protocol.Reports, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("fetching cluster status...")

	state, err := common.ReadState(ctx, stateParams)
	if err != nil {
		return
	}

	var instanceNames []string
	for _, instance := range state.Instances {
		instanceNames = append(instanceNames, instance.Name)
	}

	reports.ReadyForClusterization = instanceNames
	reports.Progress = state.Progress
	reports.Errors = state.Errors
	reports.Debug = state.Debug

	addSummary(ctx, state, stateParams, vmssParams, &reports)

	return
}

func GetClusterStatus(ctx context.Context, vmssParams *common.ScaleSetParams, stateParams common.BlobObjParams, keyVaultUri string) (clusterStatus protocol.ClusterStatus, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("fetching cluster status...")

	state, err := common.ReadState(ctx, stateParams)
	if err != nil {
		return
	}
	clusterStatus.InitialSize = state.InitialSize
	clusterStatus.DesiredSize = state.DesiredSize
	clusterStatus.Clusterized = state.Clusterized
	if !state.Clusterized {
		return
	}

	credentials, err := common.GetWekaClusterCredentials(ctx, keyVaultUri)
	if err != nil {
		return
	}

	jrpcBuilder := func(ip string) *jrpc.BaseClient {
		return connectors.NewJrpcClient(ctx, ip, weka.ManagementJrpcPort, credentials.Username, credentials.Password)
	}

	vmIps, err := common.GetVmsPrivateIps(ctx, vmssParams)
	if err != nil {
		return
	}
	ips := make([]string, 0, len(vmIps))
	for _, ip := range vmIps {
		ips = append(ips, ip)
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(ips), func(i, j int) { ips[i], ips[j] = ips[j], ips[i] })
	logger.Info().Msgf("ips: %s", ips)
	jpool := &jrpc.Pool{
		Ips:     ips,
		Clients: map[string]*jrpc.BaseClient{},
		Active:  "",
		Builder: jrpcBuilder,
		Ctx:     ctx,
	}

	var rawWekaStatus json.RawMessage

	err = jpool.Call(weka.JrpcStatus, struct{}{}, &rawWekaStatus)
	if err != nil {
		return
	}

	wekaStatus := protocol.WekaStatus{}
	if err = json.Unmarshal(rawWekaStatus, &wekaStatus); err != nil {
		return
	}
	clusterStatus.WekaStatus = wekaStatus

	return
}

func GetRefreshStatus(ctx context.Context, vmssParams *common.ScaleSetParams, stateParams common.BlobObjParams, extended bool) (*common.VMSSStateVerbose, error) {
	vmssConfig, err := common.ReadVmssConfig(ctx, stateParams.StorageName, stateParams.ContainerName)
	if err != nil {
		return nil, err
	}
	vmssConfig.CustomData = "<hidden>"
	vmssConfig.SshPublicKey = "<hidden>"

	currentConfig, err := common.GetCurrentScaleSetConfiguration(ctx, vmssParams)
	if err != nil {
		return nil, err
	}

	result := &common.VMSSStateVerbose{
		VmssName:         vmssParams.ScaleSetName,
		TargetConfig:     vmssConfig,
		TargetConfigHash: vmssConfig.ConfigHash,
	}

	if currentConfig != nil {
		result.CurrentConfigHash = currentConfig.ConfigHash
		result.NeedUpdate = vmssConfig.ConfigHash != currentConfig.ConfigHash
	}

	if extended && currentConfig != nil {
		currentConfig.CustomData = "<hidden>"
		currentConfig.SshPublicKey = "<hidden>"
		result.CurrentConfig = currentConfig
	}

	state, err := common.ReadState(ctx, stateParams)
	if err != nil {
		return nil, err
	}
	if len(state.Updates) > 0 {
		result.UpdatesLog = make([]protocol.Update, 0, len(state.Updates))
	}
	for _, update := range state.Updates {
		result.UpdatesLog = append(result.UpdatesLog, update)
	}
	return result, nil
}

func Handler(w http.ResponseWriter, r *http.Request) {
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	stateBlobName := os.Getenv("STATE_BLOB_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")
	nfsStateContainerName := os.Getenv("NFS_STATE_CONTAINER_NAME")
	nfsStateBlobName := os.Getenv("NFS_STATE_BLOB_NAME")
	nfsScaleSetName := os.Getenv("NFS_VMSS_NAME")

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	var invokeRequest common.InvokeRequest

	var requestBody struct {
		Type     string `json:"type"`
		Protocol string `json:"protocol"`
	}

	if err := json.NewDecoder(r.Body).Decode(&invokeRequest); err != nil {
		err = fmt.Errorf("cannot decode the request: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	var reqData map[string]interface{}
	err := json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		err = fmt.Errorf("cannot unmarshal the request data: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	if reqData["Body"] != nil {
		if err := json.Unmarshal([]byte(reqData["Body"].(string)), &requestBody); err != nil {
			err = fmt.Errorf("cannot unmarshal the request body: %v", err)
			logger.Error().Err(err).Send()
			common.WriteErrorResponse(w, err)
			return
		}
	}

	stateParams := common.BlobObjParams{
		StorageName:   stateStorageName,
		ContainerName: stateContainerName,
		BlobName:      stateBlobName,
	}

	vmssParams := &common.ScaleSetParams{
		SubscriptionId:    subscriptionId,
		ResourceGroupName: resourceGroupName,
		ScaleSetName:      common.GetVmScaleSetName(prefix, clusterName),
		Flexible:          false,
	}

	if requestBody.Protocol == "nfs" {
		stateParams.ContainerName = nfsStateContainerName
		stateParams.BlobName = nfsStateBlobName

		vmssParams.ScaleSetName = nfsScaleSetName
		vmssParams.Flexible = true
	}

	var result interface{}
	if requestBody.Type == "" || requestBody.Type == "status" {
		result, err = GetClusterStatus(ctx, vmssParams, stateParams, keyVaultUri)
	} else if requestBody.Type == "progress" {
		result, err = GetReports(ctx, stateParams, vmssParams)
	} else if requestBody.Type == "vmss" {
		result, err = GetRefreshStatus(ctx, vmssParams, stateParams, false)
	} else if requestBody.Type == "vmss-extended" {
		result, err = GetRefreshStatus(ctx, vmssParams, stateParams, true)
	} else {
		result = "Invalid status type"
	}

	if err != nil {
		common.WriteErrorResponse(w, err)
		return
	}
	common.WriteSuccessResponse(w, result)
}
