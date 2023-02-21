package resize

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"weka-deployment/common"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})

	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")

	ctx := r.Context()
	logger := common.LoggerFromCtx(ctx)

	var invokeRequest common.InvokeRequest

	var size struct {
		NewSize *int `json:"new_size"`
	}

	if err := json.NewDecoder(r.Body).Decode(&invokeRequest); err != nil {
		logger.Error().Msg("Bad request")
		return
	}

	var reqData map[string]interface{}
	err := json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		logger.Error().Msg("Bad request")
		return
	}

	if json.Unmarshal([]byte(reqData["Body"].(string)), &size) != nil {
		logger.Error().Msg("Bad request")
		return
	}
	if size.NewSize == nil {
		err := fmt.Errorf("wrong request format. 'new_size' is required")
		logger.Error().Err(err).Send()
		return
	}

	logger.Info().Msgf("The requested new size is %d", *size.NewSize)

	vmScaleSetName := fmt.Sprintf("%s-%s-vmss", prefix, clusterName)

	err = updateDesiredClusterSize(ctx, *size.NewSize, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName)
	if err != nil {
		resData["body"] = err.Error()
	} else {
		resData["body"] = fmt.Sprintf("New cluster size: %d", size.NewSize)
	}

	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}

func updateDesiredClusterSize(ctx context.Context, newSize int, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName string) error {
	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return err
	}
	oldSize := state.DesiredSize
	state.DesiredSize = newSize

	err = common.WriteState(ctx, stateStorageName, stateContainerName, state)
	if err != nil {
		err = fmt.Errorf("cannot update state to %d: %v", newSize, err)
		return err
	}

	if oldSize < newSize {
		err = common.UpdateVmScaleSetNum(ctx, subscriptionId, resourceGroupName, vmScaleSetName, int64(newSize))
		if err != nil {
			err = fmt.Errorf("cannot increase scale set %s capacity from %d to %d: %v", vmScaleSetName, oldSize, newSize, err)
			return err
		}
	}
	return nil
}
