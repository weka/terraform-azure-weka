package scale_down

import (
	"encoding/json"
	"net/http"

	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
	"github.com/weka/go-cloud-lib/scale_down"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	resData := make(map[string]interface{})

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	var info protocol.HostGroupInfoResponse
	err := json.NewDecoder(r.Body).Decode(&info)
	if err != nil {
		logger.Error().Msg("Bad request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	scaleResponse, err := scale_down.ScaleDown(ctx, info)
	if err != nil {
		resData["body"] = err.Error()
	} else {
		resData["body"] = scaleResponse
	}

	responseJson, _ := json.Marshal(resData)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
