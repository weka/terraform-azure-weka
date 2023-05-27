package protect

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"weka-deployment/common"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/weka/go-cloud-lib/logging"
)

type RequestBody struct {
	Vm string `json:"vm"`
}

func report(ctx context.Context, hostName, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, reportType, reportContent string) {
	reportObj := common.Report{Type: reportType, Hostname: hostName, Message: reportContent}
	_ = common.UpdateStateReporting(ctx, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, reportObj)
}

func setProtection(ctx context.Context, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, vmScaleSetName, instanceId, hostName string) (err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("Setting deletion protection on %s", hostName)
	counter := 0
	authSleepInterval := 2 //minutes
	for {
		err = common.SetDeletionProtection(ctx, subscriptionId, resourceGroupName, vmScaleSetName, instanceId, true)
		if err == nil {
			msg := "Deletion protection was set successfully"
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

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	var data RequestBody
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		logger.Error().Err(err).Send()
		common.RespondWithError(w, err, http.StatusBadRequest)
		return
	}

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	instanceName := strings.Split(data.Vm, ":")[0]
	hostName := strings.Split(data.Vm, ":")[1]
	instanceId := common.GetScaleSetVmIndex(instanceName)

	err = setProtection(ctx, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, vmScaleSetName, instanceId, hostName)
	if err != nil {
		logger.Error().Err(err).Send()
		common.RespondWithError(w, err, http.StatusInternalServerError)
		return
	}

	msg := "protection was set successfully"
	common.RespondWithMessage(w, msg, http.StatusOK)
}
