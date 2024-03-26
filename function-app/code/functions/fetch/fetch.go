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

func Handler(w http.ResponseWriter, r *http.Request) {
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	stateBlobName := os.Getenv("STATE_BLOB_NAME")
	nfsStateContainerName := os.Getenv("NFS_STATE_CONTAINER_NAME")
	nfsStateBlobName := os.Getenv("NFS_STATE_BLOB_NAME")
	nfsScaleSetName := os.Getenv("NFS_VMSS_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	backendsStateParams := common.BlobObjParams{
		StorageName:   stateStorageName,
		ContainerName: stateContainerName,
		BlobName:      stateBlobName,
	}

	nfsStateParams := common.BlobObjParams{
		StorageName:   stateStorageName,
		ContainerName: nfsStateContainerName,
		BlobName:      nfsStateBlobName,
	}

	instances, err := common.GetScaleSetInstancesInfo(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	wekaAdminPassword, err := common.GetWekaClusterPassword(ctx, keyVaultUri)
	if err != nil {
		err = fmt.Errorf("cannot get weka admin password: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	desiredCapacity, err := getCapacity(ctx, backendsStateParams)
	if err != nil {
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	nfsDesiredCapacity, err := getCapacity(ctx, nfsStateParams)
	if err != nil {
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	nfsInstances, err := common.GetScaleSetInstancesInfo(ctx, subscriptionId, resourceGroupName, nfsScaleSetName)
	if err != nil {
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	response := protocol.HostGroupInfoResponse{
		Username:                    common.WekaAdminUsername,
		Password:                    wekaAdminPassword,
		WekaBackendsDesiredCapacity: desiredCapacity,
		WekaBackendInstances:        instances,
		NFSBackendsDesiredCapacity:  nfsDesiredCapacity,
		NfsBackendInstances:         nfsInstances,
		BackendIps:                  getBackendIps(instances),
		Role:                        "backend",
		Version:                     1,
	}

	common.WriteSuccessResponse(w, response)
}

func getBackendIps(instances []protocol.HgInstance) (ips []string) {
	for _, inst := range instances {
		ips = append(ips, inst.PrivateIp)
	}
	return
}

func getCapacity(ctx context.Context, stateParams common.BlobObjParams) (desired int, err error) {
	state, err := common.ReadState(ctx, stateParams)
	if err != nil {
		return
	}
	desired = state.DesiredSize
	return
}
