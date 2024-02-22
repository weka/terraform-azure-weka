package transient

import (
	"encoding/json"
	"net/http"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	resData := make(map[string]interface{})
	var invokeRequest common.InvokeRequest

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	d := json.NewDecoder(r.Body)
	err := d.Decode(&invokeRequest)
	if err != nil {
		logger.Error().Msg("Bad request")
		common.WriteErrorResponse(w, err)
		return
	}

	var reqData map[string]interface{}
	err = json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		logger.Error().Msg("Bad request")
		common.WriteErrorResponse(w, err)
		return
	}

	var terminateResponse protocol.TerminatedInstancesResponse

	if err := json.Unmarshal([]byte(reqData["Body"].(string)), &terminateResponse); err != nil {
		logger.Error().Msg("Bad request")
		common.WriteErrorResponse(w, err)
		return
	}

	logger.Debug().Msgf("input: %#v", terminateResponse)
	errs := terminateResponse.TransientErrors
	if len(errs) > 0 {
		logger.Debug().Msgf("transient errors: %s", errs)
		resData["body"] = errs
	} else {
		logger.Debug().Msgf("no transient errors found")
		resData["body"] = "no errors"
	}

	common.WriteResponse(w, resData, nil)
}
