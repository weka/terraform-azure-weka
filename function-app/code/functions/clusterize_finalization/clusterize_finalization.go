package clusterize_finalization

import (
	"encoding/json"
	"net/http"
	"os"
	"weka-deployment/common"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	resData := make(map[string]interface{})

	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")

	ctx := r.Context()

	state, err := common.UpdateClusterized(ctx, subscriptionId, resourceGroupName, stateStorageName, stateContainerName)
	if err != nil {
		resData["body"] = err.Error()
	} else {
		resData["body"] = state
	}

	responseJson, _ := json.Marshal(resData)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
