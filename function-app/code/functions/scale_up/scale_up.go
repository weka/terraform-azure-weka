package scale_up

import (
	"fmt"
	"net/http"
	"os"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)
	vmScaleSetName := fmt.Sprintf("%s-%s-vmss", prefix, clusterName)

	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		logger.Error().Err(err).Send()
		common.RespondWithError(w, err, http.StatusInternalServerError)
		return
	}

	if !state.Clusterized {
		msg := "Not clusterized yet, skipping..."
		common.RespondWithMessage(w, msg, http.StatusOK)
		return
	}

	err = common.UpdateVmScaleSetNum(ctx, subscriptionId, resourceGroupName, vmScaleSetName, int64(state.DesiredSize))
	if err != nil {
		logger.Error().Err(err).Send()
		common.RespondWithError(w, err, http.StatusInternalServerError)
		return
	}

	msg := fmt.Sprintf("updated size to %d successfully", state.DesiredSize)
	common.RespondWithMessage(w, msg, http.StatusOK)
}
