package transient

import (
	"encoding/json"
	"net/http"
	"weka-deployment/common"
	"weka-deployment/protocol"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})
	var invokeRequest common.InvokeRequest

	ctx := r.Context()
	logger := common.LoggerFromCtx(ctx)

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

	var terminateResponse protocol.TerminatedInstancesResponse

	if json.Unmarshal([]byte(reqData["Body"].(string)), &terminateResponse) != nil {
		logger.Error().Msg("Bad request")
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

	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
