package status

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/weka/go-cloud-lib/connectors"
	"github.com/weka/go-cloud-lib/lib/jrpc"
	"github.com/weka/go-cloud-lib/lib/weka"
	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
	"weka-deployment/common"
)

func itemInList(item string, list []string) bool {
	for _, listItem := range list {
		if item == listItem {
			return true
		}
	}
	return false
}

func addSummary(ctx context.Context, state protocol.ClusterState, stateStorageName, stateContainerName, subscriptionId, resourceGroupName, vmScaleSetName string, reports *protocol.Reports) {
	if state.Clusterized {
		summary := protocol.ClusterizationStatusSummary{
			ClusterizationTarget: state.ClusterizationTarget,
			Clusterized:          state.Clusterized,
		}
		reports.Summary = summary
		return
	}

	vms, err := common.GetScaleSetVmsExpandedView(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		msg := fmt.Sprintf("Failed getting vms list for vmss %s: %v", vmScaleSetName, err)
		common.ReportMsg(ctx, "vmss", subscriptionId, resourceGroupName, stateContainerName, stateStorageName, "error", msg)
		return
	}
	toTerminate := common.GetUnhealthyInstancesToTerminate(ctx, vms)

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

	clusterizationInstance := ""
	if len(state.Instances) >= state.ClusterizationTarget {
		clusterizationInstance = strings.Split(state.Instances[state.ClusterizationTarget-1], ":")[1]
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

	return
}

func GetReports(ctx context.Context, stateStorageName, stateContainerName, subscriptionId, resourceGroupName, vmScaleSetName string) (reports protocol.Reports, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("fetching cluster status...")

	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}
	reports.ReadyForClusterization = state.Instances
	reports.Progress = state.Progress
	reports.Errors = state.Errors
	reports.Debug = state.Debug

	addSummary(ctx, state, stateStorageName, stateContainerName, subscriptionId, resourceGroupName, vmScaleSetName, &reports)

	return
}

func GetClusterStatus(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName, stateStorageName, stateContainerName, keyVaultUri string) (clusterStatus protocol.ClusterStatus, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("fetching cluster status...")

	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}
	clusterStatus.InitialSize = state.InitialSize
	clusterStatus.DesiredSize = state.DesiredSize
	clusterStatus.Clusterized = state.Clusterized
	if !state.Clusterized {
		return
	}

	wekaPassword, err := common.GetWekaClusterPassword(ctx, keyVaultUri)
	if err != nil {
		return
	}

	jrpcBuilder := func(ip string) *jrpc.BaseClient {
		return connectors.NewJrpcClient(ctx, ip, weka.ManagementJrpcPort, "admin", wekaPassword)
	}

	vmIps, err := common.GetVmsPrivateIps(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		return
	}
	ips := make([]string, len(vmIps))
	for _, ip := range vmIps {
		ips = append(ips, ip)
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(ips), func(i, j int) { ips[i], ips[j] = ips[j], ips[i] })
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

func Handler(w http.ResponseWriter, r *http.Request) {
	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})

	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	var invokeRequest common.InvokeRequest

	var requestBody struct {
		Type string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&invokeRequest); err != nil {
		err = fmt.Errorf("cannot decode the request: %v", err)
		logger.Error().Err(err).Send()
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var reqData map[string]interface{}
	err := json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		err = fmt.Errorf("cannot unmarshal the request data: %v", err)
		logger.Error().Err(err).Send()
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if reqData["Body"] != nil {
		if json.Unmarshal([]byte(reqData["Body"].(string)), &requestBody) != nil {
			err = fmt.Errorf("cannot unmarshal the request body: %v", err)
			logger.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)
	var result interface{}
	if requestBody.Type == "" || requestBody.Type == "status" {
		result, err = GetClusterStatus(ctx, subscriptionId, resourceGroupName, vmScaleSetName, stateStorageName, stateContainerName, keyVaultUri)
	} else if requestBody.Type == "progress" {
		result, err = GetReports(ctx, stateStorageName, stateContainerName, subscriptionId, resourceGroupName, vmScaleSetName)
	} else {
		result = "Invalid status type"
	}

	if err != nil {
		resData["body"] = err.Error()
	} else {

		resData["body"] = result
	}
	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
