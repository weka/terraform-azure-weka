package report

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"weka-deployment/common"
)

type Report struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	Hostname string `json:"hostname"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})

	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")

	ctx := r.Context()
	logger := common.LoggerFromCtx(ctx)

	var invokeRequest common.InvokeRequest

	var report Report

	if err := json.NewDecoder(r.Body).Decode(&invokeRequest); err != nil {
		err = fmt.Errorf("cannot decode the request: %v", err)
		logger.Error().Err(err).Send()
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var reqData map[string]interface{}
	err := json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		err = fmt.Errorf("cannot unmarshal the request data: %v", err)
		logger.Error().Err(err).Send()
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if json.Unmarshal([]byte(reqData["Body"].(string)), &report) != nil {
		err = fmt.Errorf("cannot unmarshal the request body: %v", err)
		logger.Error().Err(err).Send()
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger.Info().Msgf("Updating state %s with %s", report.Type, report.Message)
	err = UpdateState(ctx, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, report)

	if err != nil {
		resData["body"] = err.Error()
	} else {
		resData["body"] = fmt.Sprintf("Updated state errors successfully")
	}

	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}

func UpdateState(ctx context.Context, subscriptionId, resourceGroupName, stateContainerName, stateStorageName string, report Report) (err error) {
	logger := common.LoggerFromCtx(ctx)

	leaseId, err := common.LockContainer(ctx, subscriptionId, resourceGroupName, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	err = UpdateStateWithoutLocking(ctx, stateContainerName, stateStorageName, report)

	_, err2 := common.UnlockContainer(ctx, subscriptionId, resourceGroupName, stateStorageName, stateContainerName, leaseId)
	if err2 != nil {
		if err == nil {
			err = err2
		}
		logger.Error().Msgf("unlocking %s failed", stateStorageName)
	}
	return
}

func UpdateStateWithoutLocking(ctx context.Context, stateContainerName, stateStorageName string, report Report) (err error) {
	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}
	currentTime := time.Now().UTC().Format("15:04:05") + " UTC"
	switch report.Type {
	case "error":
		if state.Errors == nil {
			state.Errors = make(map[string][]string)
		}
		state.Errors[report.Hostname] = append(state.Errors[report.Hostname], fmt.Sprintf("%s: %s", currentTime, report.Message))
	case "progress":
		if state.Progress == nil {
			state.Progress = make(map[string][]string)
		}
		state.Progress[report.Hostname] = append(state.Progress[report.Hostname], fmt.Sprintf("%s: %s", currentTime, report.Message))
	default:
		err = fmt.Errorf("invalid type: %s", report.Type)
		return
	}

	err = common.WriteState(ctx, stateStorageName, stateContainerName, state)
	if err != nil {
		err = fmt.Errorf("failed updating state errors")
		return
	}
	return
}
