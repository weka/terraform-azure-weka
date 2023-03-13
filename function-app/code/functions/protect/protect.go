package protect

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"net/http"
	"os"
	"strings"
	"time"
	"weka-deployment/common"
)

type RequestBody struct {
	Vm string `json:"vm"`
}

func report(ctx context.Context, hostName, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, reportType, reportContent string) {
	reportObj := common.Report{Type: reportType, Hostname: hostName, Message: reportContent}
	_ = common.UpdateStateReporting(ctx, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, reportObj)
}

func setProtection(ctx context.Context, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, vmScaleSetName, instanceId, hostName string) (err error) {
	logger := common.LoggerFromCtx(ctx)
	logger.Info().Msgf("Setting deletion protection on %s", hostName)
	counter := 0
	authSleepInterval := 1 //minutes
	for {
		err = common.SetDeletionProtection(ctx, subscriptionId, resourceGroupName, vmScaleSetName, instanceId, true)
		if err == nil {
			msg := "Deletion protection was was set successfully"
			logger.Info().Msg(msg)
			report(ctx, hostName, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, "progress", msg)
			break
		}

		if protectionErr, ok := err.(*azcore.ResponseError); ok && protectionErr.ErrorCode == "AuthorizationFailed" {
			counter++
			if counter > 10 {
				break
			}
			msg := fmt.Sprintf("Setting deletion protection authorization error, going to sleep for %dM", authSleepInterval)
			logger.Info().Msg(msg)
			report(ctx, hostName, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, "progress", msg)
			time.Sleep(time.Duration(authSleepInterval) * time.Minute)
		} else {
			break
		}
	}
	if err != nil {
		logger.Error().Err(err).Send()
		report(ctx, hostName, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, "error", err.Error())
	}
	return
}

func Handler(w http.ResponseWriter, r *http.Request) {
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")

	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})
	var invokeRequest common.InvokeRequest

	ctx := r.Context()
	logger := common.LoggerFromCtx(ctx)

	d := json.NewDecoder(r.Body)
	err := d.Decode(&invokeRequest)
	if err != nil {
		logger.Error().Msg("Bad request")
		return
	}

	var reqData map[string]interface{}
	err = json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		logger.Error().Msg("Bad request")
		return
	}

	var data RequestBody

	if json.Unmarshal([]byte(reqData["Body"].(string)), &data) != nil {
		logger.Error().Msg("Bad request")
		return
	}

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	instanceName := strings.Split(data.Vm, ":")[0]
	hostName := strings.Split(data.Vm, ":")[1]
	instanceId := common.GetScaleSetVmIndex(instanceName)

	err = setProtection(ctx, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, vmScaleSetName, instanceId, hostName)

	if err != nil {
		resData["body"] = err.Error()
	} else {
		resData["body"] = "protection was set successfully"
	}
	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
