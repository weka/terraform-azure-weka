package transient

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"net/http"
	"weka-deployment/common"
	"weka-deployment/protocol"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})
	var invokeRequest common.InvokeRequest

	d := json.NewDecoder(r.Body)
	err := d.Decode(&invokeRequest)
	if err != nil {
		log.Error().Msg("Bad request")
		return
	}

	var reqData map[string]interface{}
	err = json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		log.Error().Msg("Bad request")
		return
	}

	var terminateResponse protocol.TerminatedInstancesResponse

	if json.Unmarshal([]byte(reqData["Body"].(string)), &terminateResponse) != nil {
		log.Error().Msg("Bad request")
		return
	}

	log.Debug().Msgf("input: %#v", terminateResponse)
	errs := terminateResponse.TransientErrors
	if len(errs) > 0 {
		log.Debug().Msgf("transient errors: %s", errs)
		resData["body"] = errs
	} else {
		log.Debug().Msgf("no transient errors found")
		resData["body"] = "no errors"
	}

	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
