package fetch

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
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

	vmScaleSetName := fmt.Sprintf("%s-%s-vmss", prefix, clusterName)

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	response, err := GetScaleSetInfoResponse(
		ctx, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName, keyVaultUri,
	)
	if err != nil {
		logger.Error().Err(err).Send()
		common.RespondWithError(w, err, http.StatusInternalServerError)
		return
	}

	common.RespondWithJson(w, response, http.StatusOK)
}

func GetScaleSetInfoResponse(
	ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName, keyVaultUri string,
) (scaleSetInfoResponse protocol.HostGroupInfoResponse, err error) {
	instances, err := common.GetScaleSetInstancesInfo(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		return
	}

	scaleSetInfo, err := common.GetScaleSetInfo(ctx, subscriptionId, resourceGroupName, vmScaleSetName, keyVaultUri)
	if err != nil {
		return
	}

	desiredCapacity, err := getCapacity(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	// convert []ScaleSetInstanceInfo to []HgInstance
	hgInstances := make([]protocol.HgInstance, 0)
	for _, instance := range instances {
		hgInstance := protocol.HgInstance{
			Id:        instance.Id,
			PrivateIp: instance.PrivateIp,
		}
		hgInstances = append(hgInstances, hgInstance)
	}

	scaleSetInfoResponse = protocol.HostGroupInfoResponse{
		Username:        scaleSetInfo.AdminUsername,
		Password:        scaleSetInfo.AdminPassword,
		DesiredCapacity: desiredCapacity,
		Instances:       hgInstances,
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
