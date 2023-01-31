package scale_up

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"weka-deployment/common"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})

	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")

	vmScaleSetName := fmt.Sprintf("%s-%s-vmss", prefix, clusterName)

	state, err := common.ReadState(stateStorageName, stateContainerName)
	if err != nil {
		resData["body"] = err.Error()
	} else {
		err = common.UpdateVmScaleSetNum(subscriptionId, resourceGroupName, vmScaleSetName, int64(state.InitialSize))
		if err != nil {
			resData["body"] = err.Error()
		} else {
			resData["body"] = "updated size successfully"
		}
	}

	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
