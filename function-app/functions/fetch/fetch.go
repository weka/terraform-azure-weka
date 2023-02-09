package fetch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"weka-deployment/common"
)

type ScaleSetInfoResponse struct {
	Username        string                        `json:"username"`
	Password        string                        `json:"password"`
	DesiredCapacity int                           `json:"desired_capacity"`
	Instances       []common.ScaleSetInstanceInfo `json:"instances"`
	BackendIps      []string                      `json:"backend_ips"`
	Role            string                        `json:"role"`
	Version         int                           `json:"version"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})

	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")

	vmScaleSetName := fmt.Sprintf("%s-%s-vmss", prefix, clusterName)

	response, err := getScaleSetInfoResponse(
		subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName, keyVaultUri,
	)
	if err != nil {
		resData["body"] = err.Error()
	} else {
		resData["body"] = response
	}

	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}

func getScaleSetInfoResponse(
	subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName, keyVaultUri string,
) (scaleSetInfoResponse ScaleSetInfoResponse, err error) {
	instances, err := common.GetScaleSetInstancesInfo(subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		return
	}

	scaleSetInfo, err := common.GetScaleSetInfo(subscriptionId, resourceGroupName, vmScaleSetName, keyVaultUri)
	if err != nil {
		return
	}

	desiredCapacity, err := getCapacity(stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	scaleSetInfoResponse = ScaleSetInfoResponse{
		Username:        scaleSetInfo.AdminUsername,
		Password:        scaleSetInfo.AdminPassword,
		DesiredCapacity: desiredCapacity,
		Instances:       instances,
		BackendIps:      getBackendIps(instances),
		Role:            "backend",
		Version:         1,
	}
	return
}

func getBackendIps(instances []common.ScaleSetInstanceInfo) (ips []string) {
	for _, inst := range instances {
		ips = append(ips, inst.PrivateIp)
	}
	return
}

func getCapacity(stateStorageName string, stateContainerName string) (desired int, err error) {
	state, err := common.ReadState(stateStorageName, stateContainerName)
	if err != nil {
		return
	}
	desired = state.DesiredSize
	return
}
