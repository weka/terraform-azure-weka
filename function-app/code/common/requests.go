package common

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/weka/go-cloud-lib/logging"
	"net/http"
)

func GetBody(ctx context.Context, w http.ResponseWriter, r *http.Request, requestBody interface{}) error {
	logger := logging.LoggerFromCtx(ctx)

	var invokeRequest InvokeRequest

	if err := json.NewDecoder(r.Body).Decode(&invokeRequest); err != nil {
		err = fmt.Errorf("cannot decode the request: %v", err)
		logger.Error().Err(err).Send()
		WriteErrorResponse(w, err)
		return nil
	}

	var reqData map[string]interface{}
	err := json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		err = fmt.Errorf("cannot unmarshal the request data: %v", err)
		logger.Error().Err(err).Send()
		WriteErrorResponse(w, err)
		return nil
	}

	if reqData["Body"] != nil {
		if err := json.Unmarshal([]byte(reqData["Body"].(string)), &requestBody); err != nil {
			err = fmt.Errorf("cannot unmarshal the request body: %v", err)
			logger.Error().Err(err).Send()
			WriteErrorResponse(w, err)
			return nil
		}
	}
	return nil
}
