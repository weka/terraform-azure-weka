package scale_up

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"weka-deployment/common"
)

type RequestBody struct {
	Size int64 `json:"size"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("running scale_up")
	clusterName := os.Getenv("CLUSTER_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")

	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})
	var invokeRequest common.InvokeRequest

	d := json.NewDecoder(r.Body)
	err := d.Decode(&invokeRequest)
	if err != nil {
		log.Error().Msg("Bad request")
		return
	}

	var reqData map[string]interface{}
	err = json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		log.Error().Msg("Bad request")
		return
	}

	var data RequestBody

	if json.Unmarshal([]byte(reqData["Body"].(string)), &data) != nil {
		log.Error().Msg("Bad request")
		return
	}

	vmScaleSetName := fmt.Sprintf("%s-%s-vmss", prefix, clusterName)
	err = common.UpdateVmScaleSetNum(subscriptionId, resourceGroupName, vmScaleSetName, data.Size)
	if err != nil {
		resData["body"] = err.Error()
	} else {
		resData["body"] = "updated size successfully"
	}

	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
