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
	var invokeRequest common.InvokeRequest

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	d := json.NewDecoder(r.Body)
	err := d.Decode(&invokeRequest)
	if err != nil {
		logger.Error().Msg("Bad request")
		return
	}

	var reqData map[string]interface{}
	err = json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		logger.Error().Msg("Bad request")
		return
	}

	var info protocol.HostGroupInfoResponse

	if json.Unmarshal([]byte(reqData["Body"].(string)), &info) != nil {
		logger.Error().Msg("Bad request")
		return
	}

	scaleResponse, err := scale_down.ScaleDown(ctx, info)
	if err != nil {
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}
	common.WriteSuccessResponse(w, scaleResponse)
}
