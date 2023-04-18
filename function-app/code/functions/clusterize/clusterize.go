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
	"weka-deployment/lib/clusterize"
	fd "weka-deployment/lib/functions_def"

	"github.com/weka/go-cloud-lib/logging"

	"github.com/lithammer/dedent"
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

	StateContainerName string
	StateStorageName   string
	InstallDpdk        bool

	VmName  string
	Cluster clusterize.ClusterParams
	Obs     AzureObsParams
}

type RequestBody struct {
	Vm string `json:"vm"`
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
	return fmt.Sprintf(`
#!/bin/bash
shutdown now
`)
}

func reportClusterizeError(ctx context.Context, p ClusterizationParams, err error) {
	hostName := strings.Split(p.VmName, ":")[1]
	report := common.Report{Type: "error", Hostname: hostName, Message: err.Error()}
	_ = common.UpdateStateReporting(ctx, p.SubscriptionId, p.ResourceGroupName, p.StateContainerName, p.StateStorageName, report)
}

func HandleLastClusterVm(ctx context.Context, state common.ClusterState, p ClusterizationParams) (clusterizeScript string) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("This is the last instance in the cluster, creating obs and clusterization script")

	var err error
	vmScaleSetName := common.GetVmScaleSetName(p.Prefix, p.Cluster.ClusterName)

	if p.Cluster.SetObs {
		if p.Obs.AccessKey == "" {
			p.Obs.AccessKey, err = common.CreateStorageAccount(
				ctx, p.SubscriptionId, p.ResourceGroupName, p.Obs.Name, p.Location,
			)
			if err != nil {
				clusterizeScript = GetErrorScript(err)
				return
			}

			err = common.CreateContainer(ctx, p.Obs.Name, p.Obs.ContainerName)
			if err != nil {
				clusterizeScript = GetErrorScript(err)
				return
			}
		}

		_, err = common.AssignStorageBlobDataContributorRoleToScaleSet(
			ctx, p.SubscriptionId, p.ResourceGroupName, vmScaleSetName, p.Obs.Name, p.Obs.ContainerName,
		)
		if err != nil {
			clusterizeScript = GetErrorScript(err)
			reportClusterizeError(ctx, p, err)
			return
		}
	}

	functionAppKey, err := common.GetKeyVaultValue(ctx, p.KeyVaultUri, "function-app-default-key")
	if err != nil {
		clusterizeScript = GetErrorScript(err)
		return
	}

	wekaPassword, err := common.GetWekaClusterPassword(ctx, p.KeyVaultUri)
	if err != nil {
		clusterizeScript = GetErrorScript(err)
		return
	}

	vmsPrivateIps, err := common.GetVmsPrivateIps(ctx, p.SubscriptionId, p.ResourceGroupName, vmScaleSetName)
	if err != nil {
		clusterizeScript = GetErrorScript(err)
		return
	}

	var vmNamesList []string
	// we make the ips list compatible to vmNames
	var ipsList []string
	for _, instance := range state.Instances {
		vm := strings.Split(instance, ":")
		ipsList = append(ipsList, vmsPrivateIps[vm[0]])
		vmNamesList = append(vmNamesList, vm[1])
	}

	logger.Info().Msg("Generating clusterization script")

	clusterParams := p.Cluster
	clusterParams.VMNames = vmNamesList
	clusterParams.IPs = ipsList
	clusterParams.ObsScript = GetObsScript(p.Obs)
	clusterParams.WekaPassword = wekaPassword
	clusterParams.WekaUsername = "admin"
	clusterParams.InstallDpdk = p.InstallDpdk

	baseFunctionUrl := fmt.Sprintf("https://%s-%s-function-app.azurewebsites.net/api/", p.Prefix, p.Cluster.ClusterName)
	funcDef := fd.NewFuncDef(baseFunctionUrl, functionAppKey)

	scriptGenerator := clusterize.ClusterizeScriptGenerator{
		Params:  clusterParams,
		FuncDef: funcDef,
	}
	clusterizeScript = scriptGenerator.GetClusterizeScript()

	logger.Info().Msg("Clusterization script generated")
	return
}

func Clusterize(ctx context.Context, p ClusterizationParams) (clusterizeScript string) {
	logger := logging.LoggerFromCtx(ctx)

	instanceName := strings.Split(p.VmName, ":")[0]
	instanceId := common.GetScaleSetVmIndex(instanceName)
	vmScaleSetName := common.GetVmScaleSetName(p.Prefix, p.Cluster.ClusterName)
	vmName := p.VmName

	ip, err := common.GetPublicIp(ctx, p.SubscriptionId, p.ResourceGroupName, vmScaleSetName, p.Prefix, p.Cluster.ClusterName, instanceId)
	if err != nil {
		logger.Error().Msg("Failed to fetch public ip")
	} else {
		vmName = fmt.Sprintf("%s:%s", vmName, ip)
	}

	state, err := common.AddInstanceToState(
		ctx, p.SubscriptionId, p.ResourceGroupName, p.StateStorageName, p.StateContainerName, vmName,
	)

	if err != nil {
		if _, ok := err.(*common.ShutdownRequired); ok {
			clusterizeScript = GetShutdownScript()
		} else {
			clusterizeScript = GetErrorScript(err)
		}
		return
	}

	if len(state.Instances) == p.Cluster.HostsNum {
		clusterizeScript = HandleLastClusterVm(ctx, state, p)
	} else {
		msg := fmt.Sprintf("This is instance number %d that is ready for clusterization (not last one), doing nothing.", len(state.Instances))
		logger.Info().Msgf(msg)
		clusterizeScript = dedent.Dedent(fmt.Sprintf(`
		#!/bin/bash
		echo "%s"
		`, msg))
	}

	return
}

func Handler(w http.ResponseWriter, r *http.Request) {
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	hostsNum, _ := strconv.Atoi(os.Getenv("HOSTS_NUM"))
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
	installDpdk, _ := strconv.ParseBool(os.Getenv("INSTALL_DPDK"))

	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})
	var invokeRequest common.InvokeRequest

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	d := json.NewDecoder(r.Body)
	err := d.Decode(&invokeRequest)
	if err != nil {
		logger.Error().Msg("Bad request")
		return
	}

	var reqData map[string]interface{}
	err = json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		logger.Error().Msg("Bad request")
		return
	}

	var data RequestBody

	if json.Unmarshal([]byte(reqData["Body"].(string)), &data) != nil {
		logger.Error().Msg("Bad request")
		return
	}

	params := ClusterizationParams{
		SubscriptionId:     subscriptionId,
		ResourceGroupName:  resourceGroupName,
		Location:           location,
		Prefix:             prefix,
		KeyVaultUri:        keyVaultUri,
		StateContainerName: stateContainerName,
		StateStorageName:   stateStorageName,
		VmName:             data.Vm,
		InstallDpdk:        installDpdk,
		Cluster: clusterize.ClusterParams{
			HostsNum:    hostsNum,
			ClusterName: clusterName,
			NvmesNum:    nvmesNum,
			SetObs:      setObs,
			DataProtection: clusterize.DataProtectionParams{
				StripeWidth:     stripeWidth,
				ProtectionLevel: protectionLevel,
				Hotspare:        hotspare,
			},
		},
		Obs: AzureObsParams{
			Name:              obsName,
			ContainerName:     obsContainerName,
			AccessKey:         obsAccessKey,
			TieringSsdPercent: tieringSsdPercent,
		},
	}

	if data.Vm == "" {
		msg := "Cluster name wasn't supplied"
		logger.Error().Msgf(msg)
		resData["body"] = msg
	} else {
		clusterizeScript := Clusterize(ctx, params)
		resData["body"] = clusterizeScript
	}
	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
