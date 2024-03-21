package resize

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	stateBlobName := os.Getenv("STATE_BLOB_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	nfsStateContainerName := os.Getenv("NFS_STATE_CONTAINER_NAME")
	nfsStateBlobName := os.Getenv("NFS_STATE_BLOB_NAME")
	nfsScaleSetName := os.Getenv("NFS_VMSS_NAME")

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	var invokeRequest common.InvokeRequest

	var resizeReq struct {
		Value    *int    `json:"value"`
		Protocol *string `json:"protocol"`
	}

	if err := json.NewDecoder(r.Body).Decode(&invokeRequest); err != nil {
		err = fmt.Errorf("cannot decode the request: %v", err)
		logger.Error().Err(err).Send()
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var reqData map[string]interface{}
	err := json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		err = fmt.Errorf("cannot unmarshal the request data: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	if err := json.Unmarshal([]byte(reqData["Body"].(string)), &resizeReq); err != nil {
		err = fmt.Errorf("cannot unmarshal the request body: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}
	if resizeReq.Value == nil {
		err := fmt.Errorf("wrong request format. 'new_size' is required")
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	stateParams := common.BlobObjParams{
		StorageName:   stateStorageName,
		ContainerName: stateContainerName,
		BlobName:      stateBlobName,
	}
	isNFSProtocol := resizeReq.Protocol != nil && *resizeReq.Protocol == "nfs"
	if isNFSProtocol {
		stateParams.ContainerName = nfsStateContainerName
		stateParams.BlobName = nfsStateBlobName

		vmScaleSetName = nfsScaleSetName

		logger = logger.WithStrValue("protocol", "nfs")
	}

	logger.Info().Msgf("The requested new size is %d", *resizeReq.Value)

	minCusterSize := 6
	if *resizeReq.Value < minCusterSize && !isNFSProtocol {
		err = fmt.Errorf("invalid size, minimal cluster size is %d", minCusterSize)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	err = updateDesiredClusterSize(ctx, *resizeReq.Value, subscriptionId, resourceGroupName, vmScaleSetName, stateParams)
	if err != nil {
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	msg := fmt.Sprintf("Updated the desired cluster size to %d successfully", *resizeReq.Value)
	logger.Info().Msg(msg)
	common.WriteSuccessResponse(w, msg)
}

func updateDesiredClusterSize(ctx context.Context, newSize int, subscriptionId, resourceGroupName, vmScaleSetName string, stateParams common.BlobObjParams) error {
	state, err := common.ReadState(ctx, stateParams)
	if err != nil {
		return err
	}

	if !state.Clusterized {
		err = fmt.Errorf("weka cluster is not ready (vmss: %s)", vmScaleSetName)
		logger := logging.LoggerFromCtx(ctx)
		logger.Error().Err(err).Send()
		return err
	}
	oldSize := state.DesiredSize
	state.DesiredSize = newSize

	err = common.WriteState(ctx, stateParams, state)
	if err != nil {
		err = fmt.Errorf("cannot update state to %d: %v", newSize, err)
		return err
	}

	if oldSize < newSize {
		err = common.ScaleUp(ctx, subscriptionId, resourceGroupName, vmScaleSetName, int64(newSize))
		if err != nil {
			err = fmt.Errorf("cannot increase scale set %s capacity from %d to %d: %v", vmScaleSetName, oldSize, newSize, err)
			return err
		}
	}
	return nil
}
