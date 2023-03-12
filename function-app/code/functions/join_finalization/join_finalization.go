package join_finalization

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"weka-deployment/common"
)

type RequestBody struct {
	Name string `json:"name"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
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

	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	vmScaleSetName := fmt.Sprintf("%s-%s-vmss", prefix, clusterName)

	err = common.SetDeletionProtection(ctx, subscriptionId, resourceGroupName, vmScaleSetName, common.GetScaleSetVmIndex(data.Name), true)
	if err != nil {
		resData["body"] = err.Error()
	} else {
		resData["body"] = "set protection successfully"
	}
	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
