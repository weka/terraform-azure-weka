package clusterize_finalization

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
)

type Protocol struct {
	Protocol protocol.ProtocolGW `json:"protocol"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	stateBlobName := os.Getenv("STATE_BLOB_NAME")
	nfsStateContainerName := os.Getenv("NFS_STATE_CONTAINER_NAME")
	nfsStateBlobName := os.Getenv("NFS_STATE_BLOB_NAME")

	var invokeRequest common.InvokeRequest

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	d := json.NewDecoder(r.Body)
	err := d.Decode(&invokeRequest)
	if err != nil {
		err = fmt.Errorf("cannot decode the request: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	var reqData map[string]interface{}
	err = json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		err = fmt.Errorf("cannot unmarshal the request data: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	var vmProtocol Protocol
	if err := json.Unmarshal([]byte(reqData["Body"].(string)), &vmProtocol); err != nil {
		err = fmt.Errorf("cannot unmarshal the request body: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	stateParams := common.BlobObjParams{
		StorageName:   stateStorageName,
		ContainerName: stateContainerName,
		BlobName:      stateBlobName,
	}

	if vmProtocol.Protocol == protocol.NFS {
		stateParams.ContainerName = nfsStateContainerName
		stateParams.BlobName = nfsStateBlobName

		// Add tag to all clusterized NFS instances
		state, err := common.ReadState(ctx, stateParams)
		if err != nil {
			logger.Error().Err(err).Msg("cannot read state")
			common.WriteErrorResponse(w, err)
			return
		}

		instanceNames := common.GetStateInstancesNames(state.Instances)
		logger.Info().Msgf("Adding tag %s to %d NFS instances %v", common.NfsInterfaceGroupPortKey, len(instanceNames), instanceNames)

		tags := map[string]string{
			common.NfsInterfaceGroupPortKey: common.NfsInterfaceGroupPortValue,
		}
		for _, instanceName := range instanceNames {
			name := strings.Split(instanceName, ":")[0]
			err := common.UpdateTagsOnVm(ctx, subscriptionId, resourceGroupName, name, tags)
			if err != nil {
				msg := fmt.Sprintf("cannot update tags on VM %v", err)
				common.ReportMsg(ctx, instanceName, stateParams, "error", msg)
				logger.Error().Err(err).Str("instance", instanceName).Msg("cannot update tags")
				continue
			}
		}
	}

	state, err := common.UpdateClusterized(ctx, subscriptionId, resourceGroupName, stateParams)
	if err != nil {
		common.WriteErrorResponse(w, err)
		return
	}
	common.WriteSuccessResponse(w, state)
}
