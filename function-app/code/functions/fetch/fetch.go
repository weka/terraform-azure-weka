package fetch

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
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
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	response, err := getScaleSetInfoResponse(
		ctx, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName, keyVaultUri,
	)
	if err != nil {
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}
	common.WriteSuccessResponse(w, response)
}

func getScaleSetInfoResponse(
	ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName, keyVaultUri string,
) (scaleSetInfoResponse ScaleSetInfoResponse, err error) {
	instances, err := common.GetScaleSetInstancesInfo(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		return
	}

	wekaAdminPassword, err := common.GetWekaClusterPassword(ctx, keyVaultUri)
	if err != nil {
		err = fmt.Errorf("cannot get weka admin password: %v", err)
		return
	}

	desiredCapacity, err := getCapacity(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	scaleSetInfoResponse = ScaleSetInfoResponse{
		Username:        common.WekaAdminUsername,
		Password:        wekaAdminPassword,
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

func getCapacity(ctx context.Context, stateStorageName string, stateContainerName string) (desired int, err error) {
	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}
	desired = state.DesiredSize
	return
}
