package clusterize_finalization

import (
	"encoding/json"
	"net/http"
	"os"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	state, err := common.UpdateClusterized(ctx, subscriptionId, resourceGroupName, stateStorageName, stateContainerName)
	if err != nil {
		logger.Error().Err(err).Send()
		common.RespondWithError(w, err, http.StatusInternalServerError)
		return
	}

	responseJson, _ := json.Marshal(state)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
