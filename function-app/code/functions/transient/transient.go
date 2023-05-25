package transient

import (
	"encoding/json"
	"net/http"

	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	resData := make(map[string]interface{})

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	var terminateResponse protocol.TerminatedInstancesResponse
	err := json.NewDecoder(r.Body).Decode(&terminateResponse)
	if err != nil {
		logger.Error().Msg("Bad request")
		w.WriteHeader(http.StatusBadRequest)
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

	responseJson, _ := json.Marshal(resData)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
