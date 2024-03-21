package join_finalization

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
)

type RequestBody struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
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

	var data RequestBody

	if err := json.Unmarshal([]byte(reqData["Body"].(string)), &data); err != nil {
		logger.Error().Msg("Bad request")
		common.WriteErrorResponse(w, err)
		return
	}

	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	nfsScaleSetName := os.Getenv("NFS_VMSS_NAME")

	vmssParams := &common.ScaleSetParams{
		SubscriptionId:    subscriptionId,
		ResourceGroupName: resourceGroupName,
		ScaleSetName:      common.GetVmScaleSetName(prefix, clusterName),
		Flexible:          false,
	}
	if data.Protocol == "nfs" {
		vmssParams.ScaleSetName = nfsScaleSetName
		vmssParams.Flexible = true
	}

	instanceId := common.GetScaleSetVmIndex(data.Name, vmssParams.Flexible)

	err = common.SetDeletionProtection(ctx, vmssParams, instanceId, true)
	if err != nil {
		err = fmt.Errorf("cannot set deletion protection: %w", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}
	common.WriteSuccessResponse(w, "set protection successfully")
}
