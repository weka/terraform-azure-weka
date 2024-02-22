package debug

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"weka-deployment/common"
	"weka-deployment/functions/azure_functions_def"
	clusterizeFunc "weka-deployment/functions/clusterize"

	"github.com/weka/go-cloud-lib/clusterize"
	"github.com/weka/go-cloud-lib/logging"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	clusterName := os.Getenv("CLUSTER_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	setObs, _ := strconv.ParseBool(os.Getenv("SET_OBS"))
	obsName := os.Getenv("OBS_NAME")
	obsContainerName := os.Getenv("OBS_CONTAINER_NAME")
	obsAccessKey := os.Getenv("OBS_ACCESS_KEY")
	location := os.Getenv("LOCATION")
	nvmesNum, _ := strconv.Atoi(os.Getenv("NVMES_NUM"))
	tieringSsdPercent := os.Getenv("TIERING_SSD_PERCENT")
	prefix := os.Getenv("PREFIX")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")
	// data protection-related vars
	stripeWidth, _ := strconv.Atoi(os.Getenv("STRIPE_WIDTH"))
	protectionLevel, _ := strconv.Atoi(os.Getenv("PROTECTION_LEVEL"))
	hotspare, _ := strconv.Atoi(os.Getenv("HOTSPARE"))

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	var invokeRequest common.InvokeRequest

	var function struct {
		Function *string `json:"function"`
		IpIndex  *string `json:"ip_index"`
	}

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

	if err := json.Unmarshal([]byte(reqData["Body"].(string)), &function); err != nil {
		err = fmt.Errorf("cannot unmarshal the request body: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}
	if function.Function == nil {
		err := fmt.Errorf("wrong request format. 'function' is required")
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	logger.Info().Msgf("The requested function is %s", *function.Function)
	var result interface{}

	if *function.Function == "clusterize" {
		state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
		if err != nil {
			result = clusterizeFunc.GetErrorScript(err)
		} else {
			params := clusterizeFunc.ClusterizationParams{
				SubscriptionId:     subscriptionId,
				ResourceGroupName:  resourceGroupName,
				Location:           location,
				Prefix:             prefix,
				KeyVaultUri:        keyVaultUri,
				StateContainerName: stateContainerName,
				StateStorageName:   stateStorageName,
				Cluster: clusterize.ClusterParams{
					ClusterName: clusterName,
					NvmesNum:    nvmesNum,
					DataProtection: clusterize.DataProtectionParams{
						StripeWidth:     stripeWidth,
						ProtectionLevel: protectionLevel,
						Hotspare:        hotspare,
					},
					SetObs: setObs,
				},
				Obs: clusterizeFunc.AzureObsParams{
					Name:              obsName,
					ContainerName:     obsContainerName,
					AccessKey:         obsAccessKey,
					TieringSsdPercent: tieringSsdPercent,
				},
			}
			result, err = clusterizeFunc.HandleLastClusterVm(ctx, state, params, &azure_functions_def.AzureFuncDef{})
			if err != nil {
				result = clusterizeFunc.GetErrorScript(err)
			}
		}
	} else if *function.Function == "instances" {
		expand := "instanceView"
		instances, err1 := common.GetScaleSetInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, &expand)
		if err1 != nil {
			result = err1.Error()
		} else {
			result = instances
		}
	} else if *function.Function == "interfaces" {
		interfaces, err1 := common.GetScaleSetVmsNetworkPrimaryNICs(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
		if err1 != nil {
			result = err1.Error()
		} else {
			result = interfaces
		}
	} else if *function.Function == "ip" {
		if function.Function == nil {
			err := fmt.Errorf("wrong request format. 'ip_index' is required for fucntion 'ip'")
			logger.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ips, err1 := common.GetPublicIp(ctx, subscriptionId, resourceGroupName, vmScaleSetName, prefix, clusterName, *function.IpIndex)
		if err1 != nil {
			result = err1.Error()
		} else {
			result = ips
		}
	} else {
		result = "unsupported function"
	}

	resData["body"] = result
	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
