package protect

import (
	"encoding/json"
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
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")

	resData := make(map[string]interface{})

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	var data RequestBody
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		logger.Error().Msg("Bad request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	instanceName := strings.Split(data.Vm, ":")[0]
	hostName := strings.Split(data.Vm, ":")[1]
	instanceId := common.GetScaleSetVmIndex(instanceName)

	maxAttempts := 10
	authSleepInterval := time.Minute * 2

	err = common.RetrySetDeletionProtectionAndReport(ctx, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, vmScaleSetName, instanceId, hostName, maxAttempts, authSleepInterval)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resData["body"] = err.Error()
	} else {
		resData["body"] = "protection was set successfully"
	}

	responseJson, _ := json.Marshal(resData)
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
