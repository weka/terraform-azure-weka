package report

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"weka-deployment/common"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
)

const BlobPermissionsErrorCode = "AuthorizationPermissionMismatch"

func isPermissionsMismatch(err error) bool {
	readErr, ok := err.(*azcore.ResponseError)
	return ok && readErr.ErrorCode == BlobPermissionsErrorCode
}

func UpdateStateReportingWithRetry(ctx context.Context, subscriptionId, resourceGroupName, stateContainerName, stateStorageName string, report protocol.Report) (err error) {
	logger := logging.LoggerFromCtx(ctx)
	counter := 0
	authSleepInterval := 10 //seconds
	for {
		err = common.UpdateStateReporting(ctx, stateContainerName, stateStorageName, report)
		if err == nil {
			break
		}

		if isPermissionsMismatch(err) {
			counter++
			if counter > 12 {
				break
			}
			logger.Info().Msgf("Failed updating state, going to sleep for %d seconds", authSleepInterval)
			time.Sleep(time.Duration(authSleepInterval) * time.Second)
		} else {
			break
		}
	}

	return
}

func Handler(w http.ResponseWriter, r *http.Request) {
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	var invokeRequest common.InvokeRequest

	var report protocol.Report

	if err := json.NewDecoder(r.Body).Decode(&invokeRequest); err != nil {
		err = fmt.Errorf("cannot decode the request: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	var reqData map[string]interface{}
	err := json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		err = fmt.Errorf("cannot unmarshal the request data: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	if err := json.Unmarshal([]byte(reqData["Body"].(string)), &report); err != nil {
		err = fmt.Errorf("cannot unmarshal the request body: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	logger.Info().Msgf("Updating state %s with %s", report.Type, report.Message)
	err = common.UpdateStateReporting(ctx, stateContainerName, stateStorageName, report)

	// Sometimes when we create a resource group and immediately run weka terraform deployment, the function-app
	// permissions are not fully ready when we invoke this endpoint. It results in a blob read permissions issue.
	if err != nil && isPermissionsMismatch(err) {
		progressReport := protocol.Report{
			Type:     "progress",
			Message:  fmt.Sprintf("Handled %s successfully", BlobPermissionsErrorCode),
			Hostname: report.Hostname,
		}
		err2 := UpdateStateReportingWithRetry(ctx, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, progressReport)
		if err2 == nil {
			err = common.UpdateStateReporting(ctx, stateContainerName, stateStorageName, report)
		}
	}

	if err != nil {
		common.WriteErrorResponse(w, err)
		return
	}
	common.WriteSuccessResponse(w, "The report was added successfully")
}
