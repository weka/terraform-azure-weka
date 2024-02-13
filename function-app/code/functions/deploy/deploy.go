package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"weka-deployment/common"
	"weka-deployment/functions/azure_functions_def"

	"github.com/weka/go-cloud-lib/bash_functions"
	"github.com/weka/go-cloud-lib/deploy"
	"github.com/weka/go-cloud-lib/join"
	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"

	"github.com/lithammer/dedent"
)

func getAzureInstanceNameCmd() string {
	return "curl -s -H Metadata:true --noproxy * http://169.254.169.254/metadata/instance?api-version=2021-02-01 | jq '.compute.name' | cut -c2- | rev | cut -c2- | rev"
}

func getWekaIoToken(ctx context.Context, keyVaultUri string) (token string, err error) {
	token, err = common.GetKeyVaultValue(ctx, keyVaultUri, "get-weka-io-token")
	return
}

func getFunctionKey(ctx context.Context, keyVaultUri string) (functionAppKey string, err error) {
	functionAppKey, err = common.GetKeyVaultValue(ctx, keyVaultUri, "function-app-default-key")
	return
}

func GetDeployScript(
	ctx context.Context,
	subscriptionId,
	resourceGroupName,
	stateStorageName,
	stateContainerName,
	prefix,
	clusterName,
	installUrl,
	keyVaultUri,
	proxyUrl,
	vm string,
	computeMemory string,
	computeContainerNum int,
	frontendContainerNum int,
	driveContainerNum int,
	installDpdk bool,
	nicsNum string,
	functionAppName string,
	gateways []string,
) (bashScript string, err error) {
	logger := logging.LoggerFromCtx(ctx)

	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	// create Function Definer
	functionKey, err := getFunctionKey(ctx, keyVaultUri)
	if err != nil {
		return
	}
	baseFunctionUrl := fmt.Sprintf("https://%s.azurewebsites.net/api/", functionAppName)
	funcDef := azure_functions_def.NewFuncDef(baseFunctionUrl, functionKey)

	instanceParams := protocol.BackendCoreCount{Compute: computeContainerNum, Frontend: frontendContainerNum, Drive: driveContainerNum, ComputeMemory: computeMemory}
	if err != nil {
		logger.Error().Err(err).Send()
		return "", err
	}

	// used for getting failure domain
	getHashedIpCommand := bash_functions.GetHashedPrivateIpBashCmd()

	if !state.Clusterized {
		var token string
		token, err = getWekaIoToken(ctx, keyVaultUri)
		if err != nil {
			return
		}

		deploymentParams := deploy.DeploymentParams{
			VMName:         vm,
			InstanceParams: instanceParams,
			WekaInstallUrl: installUrl,
			WekaToken:      token,
			InstallDpdk:    installDpdk,
			NicsNum:        nicsNum,
			Gateways:       gateways,
			ProxyUrl:       proxyUrl,
		}
		deployScriptGenerator := deploy.DeployScriptGenerator{
			FuncDef:          funcDef,
			Params:           deploymentParams,
			FailureDomainCmd: getHashedIpCommand,
		}
		bashScript = deployScriptGenerator.GetDeployScript()
	} else {
		wekaPassword, err := common.GetWekaClusterPassword(ctx, keyVaultUri)
		if err != nil {
			logger.Error().Err(err).Send()
			return "", err
		}

		vmScaleSetNames, err := common.GetScaleSetsNames(ctx, subscriptionId, resourceGroupName, stateStorageName, stateContainerName)
		if err != nil {
			logger.Error().Err(err).Send()
			return "", err
		}

		vmsPrivateIps, err := common.GetVmsPrivateIps(ctx, subscriptionId, resourceGroupName, vmScaleSetNames)
		if err != nil {
			logger.Error().Err(err).Send()
			return "", err
		}

		vmNameParts := strings.Split(vm, ":")
		vmName := vmNameParts[0]

		var ips []string
		for ipVmName, ip := range vmsPrivateIps {
			// exclude ip of the machine itself
			if ipVmName != vmName {
				ips = append(ips, ip)
			}
		}
		if len(ips) == 0 {
			err = fmt.Errorf("no instances found for scale sets, can't join")
			logger.Error().Err(err).Send()
			return "", err
		}

		if err != nil {
			logger.Error().Err(err).Send()
			return "", err
		}

		joinParams := join.JoinParams{
			WekaUsername:   common.WekaAdminUsername,
			WekaPassword:   wekaPassword,
			IPs:            ips,
			InstallDpdk:    installDpdk,
			InstanceParams: instanceParams,
			Gateways:       gateways,
			ProxyUrl:       proxyUrl,
		}

		scriptBase := `
		#!/bin/bash
		set -ex
		`

		joinScriptGenerator := join.JoinScriptGenerator{
			FailureDomainCmd:   getHashedIpCommand,
			GetInstanceNameCmd: getAzureInstanceNameCmd(),
			FindDrivesScript:   dedent.Dedent(common.FindDrivesScript),
			ScriptBase:         dedent.Dedent(scriptBase),
			Params:             joinParams,
			FuncDef:            funcDef,
		}
		bashScript = joinScriptGenerator.GetJoinScript(ctx)
	}
	bashScript = dedent.Dedent(bashScript)
	return
}

type RequestBody struct {
	Vm string `json:"vm"`
}

func getGateway(subnet string) string {
	ip, ipNet, _ := net.ParseCIDR(subnet)
	ip = ip.Mask(ipNet.Mask)
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
	return ip.String()
}

func getGateways(subnet string, nicsNum int) (gateways []string) {
	gateway := getGateway(subnet)
	gateways = make([]string, nicsNum)
	for i := range gateways {
		gateways[i] = gateway
	}
	return
}

func Handler(w http.ResponseWriter, r *http.Request) {
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	clusterName := os.Getenv("CLUSTER_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")
	computeMemory := os.Getenv("COMPUTE_MEMORY")
	computeContainerNum, _ := strconv.Atoi(os.Getenv("COMPUTE_CONTAINER_CORES_NUM"))
	frontendContainerNum, _ := strconv.Atoi(os.Getenv("FRONTEND_CONTAINER_CORES_NUM"))
	driveContainerNum, _ := strconv.Atoi(os.Getenv("DRIVE_CONTAINER_CORES_NUM"))
	installDpdk, _ := strconv.ParseBool(os.Getenv("INSTALL_DPDK"))
	nicsNum := os.Getenv("NICS_NUM")
	nicsNumInt, _ := strconv.Atoi(nicsNum)
	subnet := os.Getenv("SUBNET")
	functionAppName := os.Getenv("FUNCTION_APP_NAME")

	installUrl := os.Getenv("INSTALL_URL")
	proxyUrl := os.Getenv("PROXY_URL")

	var invokeRequest common.InvokeRequest

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	d := json.NewDecoder(r.Body)
	err := d.Decode(&invokeRequest)
	if err != nil {
		err = fmt.Errorf("cannot decode the request: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	var reqData map[string]interface{}
	err = json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		err = fmt.Errorf("cannot unmarshal the request data: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	var data RequestBody

	if json.Unmarshal([]byte(reqData["Body"].(string)), &data) != nil {
		err = fmt.Errorf("cannot unmarshal the request body: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	bashScript, err := GetDeployScript(
		ctx,
		subscriptionId,
		resourceGroupName,
		stateStorageName,
		stateContainerName,
		prefix,
		clusterName,
		installUrl,
		keyVaultUri,
		proxyUrl,
		data.Vm,
		computeMemory,
		computeContainerNum,
		frontendContainerNum,
		driveContainerNum,
		installDpdk,
		nicsNum,
		functionAppName,
		getGateways(subnet, nicsNumInt),
	)

	if err != nil {
		common.WriteErrorResponse(w, err)
		return
	}
	common.WriteSuccessResponse(w, bashScript)
}
