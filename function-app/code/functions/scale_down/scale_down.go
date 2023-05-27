package scale_down

import (
	"encoding/json"
	"net/http"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
	"github.com/weka/go-cloud-lib/scale_down"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	var info protocol.HostGroupInfoResponse
	err := json.NewDecoder(r.Body).Decode(&info)
	if err != nil {
		logger.Error().Err(err).Send()
		common.RespondWithError(w, err, http.StatusBadRequest)
		return
	}

	scaleResponse, err := scale_down.ScaleDown(ctx, info)
	if err != nil {
		logger.Error().Err(err).Send()
		common.RespondWithError(w, err, http.StatusInternalServerError)
		return
	}

	responseJson, _ := json.Marshal(scaleResponse)
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
