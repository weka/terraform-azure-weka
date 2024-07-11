package clusterize

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"weka-deployment/common"
	"weka-deployment/functions/azure_functions_def"

	"github.com/rs/zerolog/log"
	"github.com/weka/go-cloud-lib/join"

	"github.com/lithammer/dedent"

	"github.com/weka/go-cloud-lib/clusterize"
	cloudCommon "github.com/weka/go-cloud-lib/common"
	"github.com/weka/go-cloud-lib/functions_def"
	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
)

type AzureObsParams struct {
	Name              string
	ContainerName     string
	AccessKey         string
	TieringSsdPercent string
}

func GetObsScript(obsParams AzureObsParams) string {
	template := `
	TIERING_SSD_PERCENT=%s
	OBS_NAME=%s
	OBS_CONTAINER_NAME=%s
	OBS_BLOB_KEY=%s

	weka fs tier s3 add azure-obs --site local --obs-name default-local --obs-type AZURE --hostname $OBS_NAME.blob.core.windows.net --port 443 --bucket $OBS_CONTAINER_NAME --access-key-id $OBS_NAME --secret-key $OBS_BLOB_KEY --protocol https --auth-method AWSSignature4
	weka fs tier s3 attach default azure-obs
	tiering_percent=$(echo "$full_capacity * 100 / $TIERING_SSD_PERCENT" | bc)
	weka fs update default --total-capacity "$tiering_percent"B
	`
	return fmt.Sprintf(
		dedent.Dedent(template), obsParams.TieringSsdPercent, obsParams.Name, obsParams.ContainerName, obsParams.AccessKey,
	)
}

type ClusterizationParams struct {
	SubscriptionId    string
	ResourceGroupName string
	Location          string
	Prefix            string
	KeyVaultUri       string

	StateParams common.BlobObjParams
	InstallDpdk bool

	Vm             protocol.Vm
	Cluster        clusterize.ClusterParams
	NFSParams      protocol.NFSParams
	NFSStateParams common.BlobObjParams
	Obs            AzureObsParams

	FunctionAppName string
}

func GetErrorScript(err error) string {
	return fmt.Sprintf(`
#!/bin/bash
<<'###ERROR'
%s
###ERROR
exit 1
	`, err.Error())
}

func GetShutdownScript() string {
	s := `
	#!/bin/bash
	shutdown now
	`
	return dedent.Dedent(s)
}

func HandleLastClusterVm(ctx context.Context, state protocol.ClusterState, p ClusterizationParams, funcDef functions_def.FunctionDef) (clusterizeScript string, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("This is the last instance in the cluster, creating obs and clusterization script")

	vmScaleSetName := common.GetVmScaleSetName(p.Prefix, p.Cluster.ClusterName)

	if p.Cluster.SetObs {
		if p.Obs.AccessKey == "" {
			p.Obs.AccessKey, err = common.CreateStorageAccount(
				ctx, p.SubscriptionId, p.ResourceGroupName, p.Obs.Name, p.Location,
			)
			if err != nil {
				err = fmt.Errorf("failed to create storage account: %w", err)
				logger.Error().Err(err).Send()
				return
			}

			err = common.CreateContainer(ctx, p.Obs.Name, p.Obs.ContainerName)
			if err != nil {
				err = fmt.Errorf("failed to create container: %w", err)
				logger.Error().Err(err).Send()
				return
			}
		}
	}

	wekaPassword, err := common.GetWekaClusterPassword(ctx, p.KeyVaultUri)
	if err != nil {
		err = fmt.Errorf("failed to get weka cluster password: %w", err)
		logger.Error().Err(err).Send()
		return
	}

	vmssParams := &common.ScaleSetParams{
		SubscriptionId:    p.SubscriptionId,
		ResourceGroupName: p.ResourceGroupName,
		ScaleSetName:      vmScaleSetName,
		Flexible:          false,
	}
	vmsPrivateIps, err := common.GetVmsPrivateIps(ctx, vmssParams)
	if err != nil {
		err = fmt.Errorf("failed to get vms private ips: %w", err)
		logger.Error().Err(err).Send()
		return
	}

	var vmNamesList []string
	// we make the ips list compatible to vmNames
	var ipsList []string
	for _, instance := range state.Instances {
		vm := strings.Split(instance.Name, ":")
		ipsList = append(ipsList, vmsPrivateIps[vm[0]])
		vmNamesList = append(vmNamesList, vm[1])
	}

	logger.Info().Msg("Generating clusterization script")

	clusterParams := p.Cluster
	clusterParams.VMNames = vmNamesList
	clusterParams.IPs = ipsList
	clusterParams.ObsScript = GetObsScript(p.Obs)
	clusterParams.WekaPassword = wekaPassword
	clusterParams.WekaUsername = common.WekaAdminUsername
	clusterParams.InstallDpdk = p.InstallDpdk
	clusterParams.FindDrivesScript = common.FindDrivesScript
	clusterParams.ClusterizationTarget = state.ClusterizationTarget

	scriptGenerator := clusterize.ClusterizeScriptGenerator{
		Params:  clusterParams,
		FuncDef: funcDef,
	}
	clusterizeScript = scriptGenerator.GetClusterizeScript()

	logger.Info().Msg("Clusterization script generated")
	return
}

func Clusterize(ctx context.Context, p ClusterizationParams) (clusterizeScript string) {
	functionAppKey, err := common.GetKeyVaultValue(ctx, p.KeyVaultUri, "function-app-default-key")
	if err != nil {
		clusterizeScript = GetErrorScript(err)
		return
	}

	baseFunctionUrl := fmt.Sprintf("https://%s.azurewebsites.net/api/", p.FunctionAppName)
	funcDef := azure_functions_def.NewFuncDef(baseFunctionUrl, functionAppKey)
	reportFunction := funcDef.GetFunctionCmdDefinition(functions_def.Report)

	if p.Vm.Protocol == protocol.NFS {
		clusterizeScript, err = doNFSClusterize(ctx, p, funcDef)
	} else if p.Vm.Protocol == protocol.SMB || p.Vm.Protocol == protocol.SMBW || p.Vm.Protocol == protocol.S3 {
		clusterizeScript = "echo 'SMB / S3 clusterization is not supported'"
	} else {
		clusterizeScript, err = doClusterize(ctx, p, funcDef)
	}

	if err != nil {
		if _, ok := err.(*common.ShutdownRequired); ok {
			clusterizeScript = GetShutdownScript()
		} else {
			clusterizeScript = cloudCommon.GetErrorScript(err, reportFunction, p.Vm.Protocol)
		}
		return
	}

	return
}

func doNFSClusterize(ctx context.Context, p ClusterizationParams, funcDef functions_def.FunctionDef) (clusterizeScript string, err error) {
	nfsInterfaceGroupName := os.Getenv("NFS_INTERFACE_GROUP_NAME")
	nfsClientGroupName := os.Getenv("NFS_CLIENT_GROUP_NAME")
	nfsProtocolgwsNum, _ := strconv.Atoi(os.Getenv("NFS_PROTOCOL_GATEWAYS_NUM"))
	nfsSecondaryIpsNum, _ := strconv.Atoi(os.Getenv("NFS_SECONDARY_IPS_NUM"))
	nfsVmssName := os.Getenv("NFS_VMSS_NAME")
	backendLbIp := os.Getenv("BACKEND_LB_IP")

	state, err := common.AddInstanceToState(ctx, p.SubscriptionId, p.ResourceGroupName, p.NFSStateParams, p.Vm)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	msg := fmt.Sprintf("This (%s) is nfs instance %d/%d that is ready for joining the interface group", p.Vm.Name, len(state.Instances), nfsProtocolgwsNum)
	log.Info().Msgf(msg)
	if len(state.Instances) != nfsProtocolgwsNum {
		clusterizeScript = cloudCommon.GetScriptWithReport(msg, funcDef.GetFunctionCmdDefinition(functions_def.Report), p.Vm.Protocol)
		return
	}

	var containersUid []string
	var nicNames []string
	for _, instance := range state.Instances {
		containersUid = append(containersUid, instance.ContainerUid)
		nicNames = append(nicNames, instance.NicName)
	}

	vmssParams := &common.ScaleSetParams{
		SubscriptionId:    p.SubscriptionId,
		ResourceGroupName: p.ResourceGroupName,
		ScaleSetName:      nfsVmssName,
		Flexible:          true,
	}
	secondaryIps, err := common.GetScaleSetSecondaryIps(ctx, vmssParams)
	if err != nil {
		err = fmt.Errorf("failed to get scale set secondary ips: %w", err)
		log.Error().Err(err).Send()
		return
	}

	if len(secondaryIps) < nfsSecondaryIpsNum {
		err = fmt.Errorf("not enough secondary ips in vmss %s: %d/%d", nfsVmssName, len(secondaryIps), nfsSecondaryIpsNum)
		log.Error().Err(err).Send()
		return
	}

	nfsParams := protocol.NFSParams{
		InterfaceGroupName: nfsInterfaceGroupName,
		ClientGroupName:    nfsClientGroupName,
		SecondaryIps:       secondaryIps,
		ContainersUid:      containersUid,
		NicNames:           nicNames,
		HostsNum:           nfsProtocolgwsNum,
	}

	scriptGenerator := clusterize.ConfigureNfsScriptGenerator{
		Params:         nfsParams,
		FuncDef:        funcDef,
		LoadBalancerIP: backendLbIp,
		Name:           p.Vm.Name,
	}

	clusterizeScript = scriptGenerator.GetNFSSetupScript()
	log.Info().Msg("Clusterization script for NFS generated")
	return
}

func doClusterize(ctx context.Context, p ClusterizationParams, funcDef functions_def.FunctionDef) (clusterizeScript string, err error) {
	logger := logging.LoggerFromCtx(ctx)

	instanceName := strings.Split(p.Vm.Name, ":")[0]
	instanceId := common.GetScaleSetVmIndex(instanceName, false)
	vmScaleSetName := common.GetVmScaleSetName(p.Prefix, p.Cluster.ClusterName)

	vmssParams := &common.ScaleSetParams{
		SubscriptionId:    p.SubscriptionId,
		ResourceGroupName: p.ResourceGroupName,
		ScaleSetName:      vmScaleSetName,
		Flexible:          false,
	}

	ip, err := common.GetPublicIp(ctx, vmssParams, p.Prefix, p.Cluster.ClusterName, instanceId)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch public ip")
	} else {
		p.Vm.Name = fmt.Sprintf("%s:%s", p.Vm.Name, ip)
	}

	state, err := common.AddInstanceToState(
		ctx, p.SubscriptionId, p.ResourceGroupName, p.StateParams, p.Vm,
	)
	if err != nil {
		return
	}

	reportFunction := funcDef.GetFunctionCmdDefinition(functions_def.Report)

	if len(state.Instances) < state.ClusterizationTarget {
		msg := fmt.Sprintf("This (%s) is instance %d/%d that is ready for clusterization", p.Vm.Name, len(state.Instances), state.DesiredSize)
		logger.Info().Msgf(msg)
		clusterizeScript = cloudCommon.GetScriptWithReport(msg, reportFunction, p.Vm.Protocol)
	} else if len(state.Instances) == state.ClusterizationTarget {
		clusterizeScript, err = HandleLastClusterVm(ctx, state, p, funcDef)
		if err != nil {
			clusterizeScript = cloudCommon.GetErrorScript(err, reportFunction, p.Vm.Protocol)
		}
	} else {
		wekaPassword, err2 := common.GetWekaClusterPassword(ctx, p.KeyVaultUri)
		if err2 != nil {
			logger.Error().Err(err2).Send()
			return
		}

		vmsPrivateIps, err2 := common.GetVmsPrivateIps(ctx, vmssParams)
		if err2 != nil {
			err = fmt.Errorf("failed to get vms private ips: %w", err)
			logger.Error().Err(err).Send()
			return
		}

		var ipsList []string
		for _, instance := range state.Instances {
			vm := strings.Split(instance.Name, ":")
			ipsList = append(ipsList, vmsPrivateIps[vm[0]])
		}

		joinParams := join.JoinParams{
			WekaUsername: common.WekaAdminUsername,
			WekaPassword: wekaPassword,
			IPs:          ipsList,
		}

		joinScriptGenerator := join.JoinScriptGenerator{
			GetInstanceNameCmd: common.GetAzureInstanceNameCmd(),
			FindDrivesScript:   dedent.Dedent(common.FindDrivesScript),
			Params:             joinParams,
			FuncDef:            funcDef,
		}
		clusterizeScript = joinScriptGenerator.GetExistingContainersJoinScript(ctx)
	}
	return
}

func Handler(w http.ResponseWriter, r *http.Request) {
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	stateBlobName := os.Getenv("STATE_BLOB_NAME")
	clusterizationTarget, _ := strconv.Atoi(os.Getenv("CLUSTERIZATION_TARGET"))
	clusterName := os.Getenv("CLUSTER_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	setObs, _ := strconv.ParseBool(os.Getenv("SET_OBS"))
	createConfigFs, _ := strconv.ParseBool(os.Getenv("CREATE_CONFIG_FS"))
	obsName := os.Getenv("OBS_NAME")
	obsContainerName := os.Getenv("OBS_CONTAINER_NAME")
	obsAccessKey := os.Getenv("OBS_ACCESS_KEY")
	location := os.Getenv("LOCATION")
	tieringSsdPercent := os.Getenv("TIERING_SSD_PERCENT")
	tieringTargetSsdRetention, _ := strconv.Atoi(os.Getenv("TIERING_TARGET_SSD_RETENTION"))
	tieringStartDemote, _ := strconv.Atoi(os.Getenv("TIERING_START_DEMOTE"))
	prefix := os.Getenv("PREFIX")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")
	// data protection-related vars
	stripeWidth, _ := strconv.Atoi(os.Getenv("STRIPE_WIDTH"))
	protectionLevel, _ := strconv.Atoi(os.Getenv("PROTECTION_LEVEL"))
	hotspare, _ := strconv.Atoi(os.Getenv("HOTSPARE"))
	installDpdk, _ := strconv.ParseBool(os.Getenv("INSTALL_DPDK"))
	addFrontendNum, _ := strconv.Atoi(os.Getenv("FRONTEND_CONTAINER_CORES_NUM"))
	functionAppName := os.Getenv("FUNCTION_APP_NAME")
	proxyUrl := os.Getenv("PROXY_URL")
	wekaHomeUrl := os.Getenv("WEKA_HOME_URL")
	preStartIoScript := os.Getenv("PRE_START_IO_SCRIPT")
	postClusterCreationScript := os.Getenv("POST_CLUSTER_CREATION_SCRIPT")
	// NFS state
	nfsStateContainerName := os.Getenv("NFS_STATE_CONTAINER_NAME")
	nfsStateBlobName := os.Getenv("NFS_STATE_BLOB_NAME")

	addFrontend := false
	if addFrontendNum > 0 {
		addFrontend = true
	}

	resData := make(map[string]interface{})
	var invokeRequest common.InvokeRequest

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	d := json.NewDecoder(r.Body)
	if err := d.Decode(&invokeRequest); err != nil {
		logger.Error().Msg("Bad request")
		common.WriteErrorResponse(w, err)
		return
	}

	var reqData map[string]interface{}
	if err := json.Unmarshal(invokeRequest.Data["req"], &reqData); err != nil {
		logger.Error().Msg("Bad request")
		common.WriteErrorResponse(w, err)
		return
	}

	var vm protocol.Vm
	if err := json.Unmarshal([]byte(reqData["Body"].(string)), &vm); err != nil {
		logger.Error().Msg("Bad request")
		common.WriteErrorResponse(w, err)
		return
	}

	params := ClusterizationParams{
		SubscriptionId:    subscriptionId,
		ResourceGroupName: resourceGroupName,
		Location:          location,
		Prefix:            prefix,
		KeyVaultUri:       keyVaultUri,
		StateParams:       common.BlobObjParams{StorageName: stateStorageName, ContainerName: stateContainerName, BlobName: stateBlobName},
		Vm:                vm,
		InstallDpdk:       installDpdk,
		Cluster: clusterize.ClusterParams{
			ClusterizationTarget: clusterizationTarget,
			ClusterName:          clusterName,
			SetObs:               setObs,
			CreateConfigFs:       createConfigFs,
			AddFrontend:          addFrontend,
			ProxyUrl:             proxyUrl,
			WekaHomeUrl:          wekaHomeUrl,
			DataProtection: clusterize.DataProtectionParams{
				StripeWidth:     stripeWidth,
				ProtectionLevel: protectionLevel,
				Hotspare:        hotspare,
			},
			PreStartIoScript:          preStartIoScript,
			PostClusterCreationScript: postClusterCreationScript,
			TieringTargetSSDRetention: tieringTargetSsdRetention,
			TieringStartDemote:        tieringStartDemote,
		},
		Obs: AzureObsParams{
			Name:              obsName,
			ContainerName:     obsContainerName,
			AccessKey:         obsAccessKey,
			TieringSsdPercent: tieringSsdPercent,
		},
		NFSStateParams:  common.BlobObjParams{StorageName: stateStorageName, ContainerName: nfsStateContainerName, BlobName: nfsStateBlobName},
		FunctionAppName: functionAppName,
	}

	status := http.StatusOK
	if vm.Name == "" {
		msg := "Cluster name wasn't supplied"
		logger.Error().Msgf(msg)
		resData["body"] = msg
		status = http.StatusBadRequest
	} else {
		clusterizeScript := Clusterize(ctx, params)
		resData["body"] = clusterizeScript
	}
	common.WriteResponse(w, resData, &status)
}
