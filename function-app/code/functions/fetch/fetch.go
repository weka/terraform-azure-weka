package fetch

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/lib/types"
	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
)

const defaultDownBackendsRemovalTimeout = 30 * time.Minute

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
	downBackendsRemovalTimeout, _ := time.ParseDuration(os.Getenv("DOWN_BACKENDS_REMOVAL_TIMEOUT"))

	if downBackendsRemovalTimeout == 0 {
		downBackendsRemovalTimeout = defaultDownBackendsRemovalTimeout
	}

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	backendsStateParams := common.BlobObjParams{
		StorageName:   stateStorageName,
		ContainerName: stateContainerName,
		BlobName:      stateBlobName,
	}

	vmssParams := &common.ScaleSetParams{
		SubscriptionId:    subscriptionId,
		ResourceGroupName: resourceGroupName,
		ScaleSetName:      vmScaleSetName,
		Flexible:          false,
	}

	instances, err := common.GetScaleSetInstancesInfo(ctx, vmssParams)
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

	response := protocol.HostGroupInfoResponse{
		Username:                    common.WekaAdminUsername,
		Password:                    wekaAdminPassword,
		WekaBackendsDesiredCapacity: desiredCapacity,
		WekaBackendInstances:        instances,
		DownBackendsRemovalTimeout:  downBackendsRemovalTimeout,
		BackendIps:                  getBackendIps(instances),
		Role:                        "backend",
		Version:                     1,
	}

	if nfsScaleSetName != "" {
		nfsStateParams := common.BlobObjParams{
			StorageName:   stateStorageName,
			ContainerName: nfsStateContainerName,
			BlobName:      nfsStateBlobName,
		}

		nfsVmssParams := &common.ScaleSetParams{
			SubscriptionId:    subscriptionId,
			ResourceGroupName: resourceGroupName,
			ScaleSetName:      nfsScaleSetName,
			Flexible:          true,
		}

		nfsDesiredCapacity, err := getCapacity(ctx, nfsStateParams)
		if err != nil {
			logger.Error().Err(err).Send()
			common.WriteErrorResponse(w, err)
			return
		}

		nfsVms, err := common.GetScaleSetInstances(ctx, nfsVmssParams)
		if err != nil {
			logger.Error().Err(err).Send()
			common.WriteErrorResponse(w, err)
			return
		}

		nfsInstancesInfo, err := common.GetScaleSetInstancesInfoFromVms(ctx, nfsVmssParams, nfsVms)
		if err != nil {
			logger.Error().Err(err).Send()
			common.WriteErrorResponse(w, err)
			return
		}

		response.NfsBackendsDesiredCapacity = nfsDesiredCapacity
		response.NfsBackendInstances = nfsInstancesInfo
		response.NfsInterfaceGroupInstanceIps = getInterfaceGroupInstanceIps(nfsVms, nfsInstancesInfo)
	}

	common.WriteSuccessResponse(w, response)
}

func getInterfaceGroupInstanceIps(vms []*common.VMInfoSummary, instancesInfo []protocol.HgInstance) (nfsInterfaceGroupInstanceIps map[string]types.Nilt) {
	vmIdsToPrivateIps := make(map[string]string, len(instancesInfo))
	for _, inst := range instancesInfo {
		vmIdsToPrivateIps[inst.Id] = inst.PrivateIp
	}

	nfsInterfaceGroupInstanceIps = make(map[string]types.Nilt)
	for _, vm := range vms {
		for key, val := range vm.Tags {
			if key == common.NfsInterfaceGroupPortKey && val != nil && *val == common.NfsInterfaceGroupPortValue {
				privateIp, ok := vmIdsToPrivateIps[common.GetScaleSetVmId(vm.ID)]
				if ok {
					nfsInterfaceGroupInstanceIps[privateIp] = types.Nilt{}
				}
			}
		}
	}
	return
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
