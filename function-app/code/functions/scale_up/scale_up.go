package scale_up

import (
	"encoding/json"
	"fmt"
	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
	"net/http"
	"os"
	"strings"
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

func Handler(w http.ResponseWriter, r *http.Request) {
	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})

	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")

	ctx := r.Context()
	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)
	logger := logging.LoggerFromCtx(ctx)
	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		resData["body"] = err.Error()
	} else {
		desiredSize := state.InitialSize
		msg := fmt.Sprintf("Not clusterized yet, initial size: %d is set", desiredSize)
		if state.Clusterized {
			desiredSize = state.DesiredSize
			msg = "updated size successfully"
		} else {
			vms, err1 := common.GetScaleSetVmsExpandedView(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
			if err1 != nil {
				msg = fmt.Sprintf("Failed getting vms list for vmss %s: %v", vmScaleSetName, err1)
				common.ReportMsg(ctx, "vmss", subscriptionId, resourceGroupName, stateContainerName, stateStorageName, "error", msg)
			}
			toTerminate := common.GetUnhealthyInstancesToTerminate(ctx, vms)
			if len(toTerminate) > 0 {
				msg = fmt.Sprintf("Terminating unhealthy instances indexes: %v", toTerminate)
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
				msg = fmt.Sprintf("errors during terminating unhealthy instances: %v", terminateErrors)
				logger.Info().Msgf(msg)
				common.ReportMsg(ctx, "vmss", subscriptionId, resourceGroupName, stateContainerName, stateStorageName, "error", msg)
			}
		}
		err = common.ScaleUp(ctx, subscriptionId, resourceGroupName, vmScaleSetName, int64(desiredSize))
		if err != nil {
			resData["body"] = err.Error()
		} else {
			resData["body"] = msg
		}
	}

	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
