package debug

import (
	"encoding/json"
	"net/http"
	"os"
	"weka-deployment/common"
	"weka-deployment/functions/clusterize"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	hostsNum := os.Getenv("HOSTS_NUM")
	clusterName := os.Getenv("CLUSTER_NAME")
	computeMemory := os.Getenv("COMPUTE_MEMORY")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	setObs := os.Getenv("SET_OBS")
	obsName := os.Getenv("OBS_NAME")
	obsContainerName := os.Getenv("OBS_CONTAINER_NAME")
	obsAccessKey := os.Getenv("OBS_ACCESS_KEY")
	location := os.Getenv("LOCATION")
	drivesContainerNum := os.Getenv("NUM_DRIVE_CONTAINERS")
	computeContainerNum := os.Getenv("NUM_COMPUTE_CONTAINERS")
	frontendContainerNum := os.Getenv("NUM_FRONTEND_CONTAINERS")
	tieringSsdPercent := os.Getenv("TIERING_SSD_PERCENT")
	prefix := os.Getenv("PREFIX")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")

	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})

	state, err := common.ReadState(stateStorageName, stateContainerName)
	var clusterizeScript string
	if err != nil {
		clusterizeScript = clusterize.GetErrorScript(err)
	} else {
		clusterizeScript = clusterize.HandleLastClusterVm(
			state, hostsNum, clusterName, computeMemory, subscriptionId,
			resourceGroupName, setObs, obsName, obsContainerName, obsAccessKey, location, drivesContainerNum,
			computeContainerNum, frontendContainerNum, tieringSsdPercent, prefix, keyVaultUri)
	}

	resData["body"] = clusterizeScript
	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
