package transient

import (
	"encoding/json"
	"net/http"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	var terminateResponse protocol.TerminatedInstancesResponse
	err := json.NewDecoder(r.Body).Decode(&terminateResponse)
	if err != nil {
		logger.Error().Err(err).Send()
		common.RespondWithError(w, err, http.StatusBadRequest)
		return
	}

	logger.Debug().Msgf("input: %#v", terminateResponse)
	errs := terminateResponse.TransientErrors
	if len(errs) == 0 {
		msg := "no transient errors found"
		logger.Debug().Msg(msg)
		common.RespondWithMessage(w, msg, http.StatusOK)
		return
	}

	logger.Debug().Msgf("transient errors: %s", errs)
	resData := make(map[string]interface{})
	resData["errors"] = errs

	common.RespondWithJson(w, resData, http.StatusOK)
}
