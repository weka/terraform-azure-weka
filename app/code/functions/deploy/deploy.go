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

func GetBackendCoreCount(instanceType string) (backendCoreCount protocol.BackendCoreCount, err error) {
	switch instanceType {
	case "Standard_L8s_v3":
		backendCoreCount = protocol.BackendCoreCount{Total: 3, Frontend: 1, Drive: 1, Memory: "31GB"}
	case "Standard_L16s_v3":
		backendCoreCount = protocol.BackendCoreCount{Total: 7, Frontend: 1, Drive: 2, Memory: "72GB"}
	case "Standard_L32s_v3":
		backendCoreCount = protocol.BackendCoreCount{Total: 7, Frontend: 1, Drive: 2, Memory: "189GB"}
	case "Standard_L48s_v3":
		backendCoreCount = protocol.BackendCoreCount{Total: 7, Frontend: 1, Drive: 3, Memory: "306GB"}
	case "Standard_L64s_v3":
		backendCoreCount = protocol.BackendCoreCount{Total: 7, Frontend: 1, Drive: 2, Memory: "418GB"}
	default:
		err = fmt.Errorf("unsupported instance type: %s", instanceType)
	}
	return backendCoreCount, err
}

func getWekaIoToken(ctx context.Context, keyVaultUri string) (token string, err error) {
	token, err = common.GetKeyVaultValue(ctx, keyVaultUri, "get-weka-io-token")
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
	instanceType,
	installUrl,
	keyVaultUri,
	vm string,
	computeMemory string,
	computeContainerNum string,
	frontendContainerNum string,
	driveContainerNum string,
	installDpdk bool,
	nicsNum string,
	gateways []string,
) (bashScript string, err error) {
	logger := logging.LoggerFromCtx(ctx)

	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	funcDef := azure_functions_def.NewFuncDef()

	// used for getting failure domain
	getHashedIpCommand := bash_functions.GetHashedPrivateIpBashCmd()

	if !state.Clusterized {
		var token string
		token, err = getWekaIoToken(ctx, keyVaultUri)
		if err != nil {
			return
		}

		deploymentParams := deploy.DeploymentParams{
			VMName:               vm,
			ComputeMemory:        computeMemory,
			ComputeContainerNum:  computeContainerNum,
			FrontendContainerNum: frontendContainerNum,
			DriveContainerNum:    driveContainerNum,
			WekaInstallUrl:       installUrl,
			WekaToken:            token,
			InstallDpdk:          installDpdk,
			NicsNum:              nicsNum,
			Gateways:             gateways,
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

		vmScaleSetName := fmt.Sprintf("%s-%s-vmss", prefix, clusterName)
		vmsPrivateIps, err := common.GetVmsPrivateIps(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
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
			err = fmt.Errorf("no instances found for instance group %s, can't join", vmScaleSetName)
			logger.Error().Err(err).Send()
			return "", err
		}

		instanceParams, err := GetBackendCoreCount(instanceType)
		if err != nil {
			logger.Error().Err(err).Send()
			return "", err
		}

		joinParams := join.JoinParams{
			WekaUsername:   "admin",
			WekaPassword:   wekaPassword,
			IPs:            ips,
			InstallDpdk:    installDpdk,
			InstanceParams: instanceParams,
			Gateways:       gateways,
		}

		scriptBase := `
		#!/bin/bash
		set -ex
		`

		findDrivesScript := `
		import json
		import sys
		for d in json.load(sys.stdin)['disks']:
			if d['isRotational']: continue
			print(d['devPath'])
		`

		joinScriptGenerator := join.JoinScriptGenerator{
			FailureDomainCmd:   getHashedIpCommand,
			GetInstanceNameCmd: getAzureInstanceNameCmd(),
			FindDrivesScript:   dedent.Dedent(findDrivesScript),
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
	computeContainerNum := os.Getenv("NUM_COMPUTE_CONTAINERS")
	frontendContainerNum := os.Getenv("NUM_FRONTEND_CONTAINERS")
	driveContainerNum := os.Getenv("NUM_DRIVE_CONTAINERS")
	installDpdk, _ := strconv.ParseBool(os.Getenv("INSTALL_DPDK"))
	nicsNum := os.Getenv("NICS_NUM")
	nicsNumInt, _ := strconv.Atoi(nicsNum)
	subnet := os.Getenv("SUBNET")

	instanceType := os.Getenv("INSTANCE_TYPE")
	installUrl := os.Getenv("INSTALL_URL")

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	var data RequestBody
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		err = fmt.Errorf("cannot decode the request: %v", err)
		logger.Error().Err(err).Send()
		common.RespondWithError(w, err, http.StatusBadRequest)
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
		instanceType,
		installUrl,
		keyVaultUri,
		data.Vm,
		computeMemory,
		computeContainerNum,
		frontendContainerNum,
		driveContainerNum,
		installDpdk,
		nicsNum,
		getGateways(subnet, nicsNumInt),
	)

	if err != nil {
		logger.Error().Err(err).Send()
		common.RespondWithError(w, err, http.StatusInternalServerError)
		return
	}
	common.RespondWithPlainText(w, bashScript, http.StatusOK)
}
