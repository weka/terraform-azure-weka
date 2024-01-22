package join_finalization

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
)

type RequestBody struct {
	Name string `json:"name"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
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

	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	clusterName := os.Getenv("CLUSTER_NAME")

	vmScaleSetName, err := common.GetScaleSetNameWithLatestVersion(ctx, subscriptionId, resourceGroupName, clusterName)
	if err != nil {
		err = fmt.Errorf("cannot get scale set with latest version: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	err = common.SetDeletionProtection(ctx, subscriptionId, resourceGroupName, vmScaleSetName, common.GetScaleSetVmIndex(data.Name), true)
	if err != nil {
		err = fmt.Errorf("cannot set deletion protection: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}
	common.WriteSuccessResponse(w, "set protection successfully")
}
