package protect

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
)

type RequestBody struct {
	Vm string `json:"vm"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	clusterName := os.Getenv("CLUSTER_NAME")

	var invokeRequest common.InvokeRequest

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	d := json.NewDecoder(r.Body)
	err := d.Decode(&invokeRequest)
	if err != nil {
		logger.Error().Msg("Bad request")
		common.WriteErrorResponse(w, err)
		return
	}

	var reqData map[string]interface{}
	err = json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		logger.Error().Msg("Bad request")
		common.WriteErrorResponse(w, err)
		return
	}

	var data RequestBody

	if json.Unmarshal([]byte(reqData["Body"].(string)), &data) != nil {
		logger.Error().Msg("Bad request")
		common.WriteErrorResponse(w, err)
		return
	}

	vmScaleSetName, err := common.GetScaleSetNameWithLatestVersion(ctx, subscriptionId, resourceGroupName, clusterName)
	if err != nil {
		err = fmt.Errorf("cannot get scale set with latest version: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	instanceName := strings.Split(data.Vm, ":")[0]
	hostName := strings.Split(data.Vm, ":")[1]
	instanceId := common.GetScaleSetVmIndex(instanceName)

	maxAttempts := 10
	authSleepInterval := time.Minute * 2

	err = common.RetrySetDeletionProtectionAndReport(ctx, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, vmScaleSetName, instanceId, hostName, maxAttempts, authSleepInterval)
	if err != nil {
		common.WriteErrorResponse(w, err)
		return
	}
	common.WriteSuccessResponse(w, "protection was set successfully")
}
