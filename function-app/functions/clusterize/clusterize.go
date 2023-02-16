package clusterize

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"weka-deployment/common"

	"github.com/lithammer/dedent"
	"github.com/rs/zerolog/log"
)

type ObsParams struct {
	SetObs        string
	Name          string
	ContainerName string
	AccessKey     string
}

type DataProtectionParams struct {
	StripeWidth     int
	ProtectionLevel int
	Hotspare        int
}

type WekaClusterParams struct {
	VmName               string
	HostsNum             string
	Name                 string
	ComputeMemory        string
	DrivesContainerNum   string
	ComputeContainerNum  string
	FrontendContainerNum string
	TieringSsdPercent    string
	DataProtection       DataProtectionParams
}

type ClusterizationParams struct {
	SubscriptionId    string
	ResourceGroupName string
	Location          string
	Prefix            string
	KeyVaultUri       string

	StateContainerName string
	StateStorageName   string

	Cluster WekaClusterParams
	Obs     ObsParams
}

type RequestBody struct {
	Vm string `json:"vm"`
}

func generateClusterizationScript(
	vmNames, ips, prefix, functionAppKey, wekaPassword string, cluster WekaClusterParams, obs ObsParams,
) (clusterizeScript string) {
	log.Info().Msg("Generating clusterization script")
	clusterizeScriptTemplate := `
	#!/bin/bash
	
	set -ex
	VMS="%s"
	IPS=%s
	HOSTS_NUM=%s
	NUM_DRIVE_CONTAINERS=%s
	CLUSTER_NAME=%s
	NUM_COMPUTE_CONTAINERS=%s
	COMPUTE_MEMORY=%s
	NUM_FRONTEND_CONTAINERS=%s
	SET_OBS=%s
	OBS_NAME=%s
	OBS_CONTAINER_NAME=%s
	OBS_BLOB_KEY=%s
	TIERING_SSD_PERCENT=%s
	PREFIX=%s
	FUNCTION_APP_KEY="%s"
	STRIPE_WIDTH=%d
	PROTECTION_LEVEL=%d
	HOTSPARE=%d
	WEKA_PASSWORD="%s"

	weka_status_ready="Containers: 1/1 running (1 weka)"
	ssh_command="ssh -o StrictHostKeyChecking=no"
	
	weka cluster create $VMS --host-ips $IPS --admin-password "$WEKA_PASSWORD"
	weka user login admin "$WEKA_PASSWORD"
	
	sleep 30s
	
	for (( i=0; i<$HOSTS_NUM; i++ )); do
		for (( d=0; d<$NUM_DRIVE_CONTAINERS; d++ )); do
			weka cluster drive add $i "/dev/nvme$d"n1
		done
	done

	weka cluster update --cluster-name="$CLUSTER_NAME"
	
	for vm in $VMS; do
	  $ssh_command $vm "sudo weka local setup container --name compute0 --base-port 15000 --cores $NUM_COMPUTE_CONTAINERS --no-frontends --compute-dedicated-cores $NUM_COMPUTE_CONTAINERS  --memory $COMPUTE_MEMORY --join-ips $IPS"
	done
	
	weka cloud enable
	weka cluster update --data-drives $STRIPE_WIDTH --parity-drives $PROTECTION_LEVEL
	weka cluster hot-spare $HOTSPARE
	weka cluster start-io
	
	for vm in $VMS; do
	  $ssh_command $vm "sudo weka local setup container --name frontend0 --base-port 16000 --cores $NUM_FRONTEND_CONTAINERS --frontend-dedicated-cores $NUM_FRONTEND_CONTAINERS --allow-protocols true --join-ips $IPS"
	done
	
	sleep 15s
	
	weka cluster process
	weka cluster drive
	weka cluster container
	
	full_capacity=$(weka status -J | jq .capacity.unprovisioned_bytes)
	weka fs group create default
	weka fs create default default "$full_capacity"B
	
	if [[ $SET_OBS == true ]]; then
	  weka fs tier s3 add azure-obs --site local --obs-name default-local --obs-type AZURE --hostname $OBS_NAME.blob.core.windows.net --port 443 --bucket $OBS_CONTAINER_NAME --access-key-id $OBS_NAME --secret-key $OBS_BLOB_KEY --protocol https --auth-method AWSSignature4
	  weka fs tier s3 attach default azure-obs
	  tiering_percent=$(echo "$full_capacity * 100 / $TIERING_SSD_PERCENT" | bc)
	  weka fs update default --total-capacity "$tiering_percent"B
	fi
	
	weka alerts mute JumboConnectivity 365d
	weka alerts mute UdpModePerformanceWarning 365d

	echo "completed successfully" > /tmp/weka_clusterization_completion_validation

	curl "https://$PREFIX-$CLUSTER_NAME-function-app.azurewebsites.net/api/clusterize_finalization?code=$FUNCTION_APP_KEY"
	`

	log.Info().Msgf("Formatting clusterization script template")
	clusterizeScript = fmt.Sprintf(
		dedent.Dedent(clusterizeScriptTemplate), vmNames, ips, cluster.HostsNum, cluster.DrivesContainerNum,
		cluster.Name, cluster.ComputeContainerNum, cluster.ComputeMemory, cluster.FrontendContainerNum,
		obs.SetObs, obs.Name, obs.ContainerName, obs.AccessKey, cluster.TieringSsdPercent, prefix, functionAppKey,
		cluster.DataProtection.StripeWidth, cluster.DataProtection.ProtectionLevel, cluster.DataProtection.Hotspare,
		wekaPassword,
	)
	return
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

func HandleLastClusterVm(state common.ClusterState, p ClusterizationParams) (clusterizeScript string) {
	log.Info().Msg("This is the last instance in the cluster, creating obs and clusterization script")

	var err error
	if p.Obs.SetObs == "true" && p.Obs.AccessKey == "" {
		p.Obs.AccessKey, err = common.CreateStorageAccount(
			p.SubscriptionId, p.ResourceGroupName, p.Obs.Name, p.Location,
		)
		if err != nil {
			clusterizeScript = GetErrorScript(err)
			return
		}

		err = common.CreateContainer(p.Obs.Name, p.Obs.ContainerName)
		if err != nil {
			clusterizeScript = GetErrorScript(err)
			return
		}
	}

	functionAppKey, err := common.GetKeyVaultValue(p.KeyVaultUri, "function-app-default-key")
	if err != nil {
		clusterizeScript = GetErrorScript(err)
		return
	}

	wekaPassword, err := common.GetWekaClusterPassword(p.KeyVaultUri)
	if err != nil {
		clusterizeScript = GetErrorScript(err)
		return
	}

	vmScaleSetName := common.GetVmScaleSetName(p.Prefix, p.Cluster.Name)
	vmsPrivateIps, err := common.GetVmsPrivateIps(p.SubscriptionId, p.ResourceGroupName, vmScaleSetName)
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
	vmNames := strings.Join(vmNamesList, " ")
	ips := strings.Join(ipsList, ",")

	clusterizeScript = generateClusterizationScript(vmNames, ips, p.Prefix, functionAppKey, wekaPassword, p.Cluster, p.Obs)
	return
}

func Clusterize(p ClusterizationParams) (clusterizeScript string) {
	state, err := common.AddInstanceToState(
		p.SubscriptionId, p.ResourceGroupName, p.StateStorageName, p.StateContainerName, p.Cluster.VmName,
	)

	if err != nil {
		if _, ok := err.(*common.ShutdownRequired); ok {
			clusterizeScript = GetShutdownScript()
		} else {
			clusterizeScript = GetErrorScript(err)
		}
		return
	}

	initialSize, err := strconv.Atoi(p.Cluster.HostsNum)
	if err != nil {
		clusterizeScript = GetErrorScript(err)
		return
	}

	vmScaleSetName := common.GetVmScaleSetName(p.Prefix, p.Cluster.Name)
	instanceName := strings.Split(p.Cluster.VmName, ":")[0]
	err = common.SetDeletionProtection(p.SubscriptionId, p.ResourceGroupName, vmScaleSetName, instanceName, true)
	if err != nil {
		clusterizeScript = GetErrorScript(err)
		return
	}

	if len(state.Instances) == initialSize {
		clusterizeScript = HandleLastClusterVm(state, p)
	} else {
		msg := fmt.Sprintf("This is instance number %d that is ready for clusterization (not last one), doing nothing.", len(state.Instances))
		log.Info().Msgf(msg)
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
	hostsNum := os.Getenv("HOSTS_NUM")
	clusterName := os.Getenv("CLUSTER_NAME")
	computeMemory := os.Getenv("COMPUTE_MEMORY")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	setObs := os.Getenv("SET_OBS")
	obsName := os.Getenv("OBS_NAME")
	obsContainerName := os.Getenv("OBS_CONTAINER_NAME")
	obsAccessKey := os.Getenv("OBS_ACCESS_KEY")
	location := os.Getenv("LOCATION")
	drivesContainerNum := os.Getenv("NUM_DRIVE_CONTAINERS")
	computeContainerNum := os.Getenv("NUM_COMPUTE_CONTAINERS")
	frontendContainerNum := os.Getenv("NUM_FRONTEND_CONTAINERS")
	tieringSsdPercent := os.Getenv("TIERING_SSD_PERCENT")
	prefix := os.Getenv("PREFIX")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")
	// data protection-related vars
	stripeWidth, _ := strconv.Atoi(os.Getenv("STRIPE_WIDTH"))
	protectionLevel, _ := strconv.Atoi(os.Getenv("PROTECTION_LEVEL"))
	hotspare, _ := strconv.Atoi(os.Getenv("HOTSPARE"))

	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})
	var invokeRequest common.InvokeRequest

	d := json.NewDecoder(r.Body)
	err := d.Decode(&invokeRequest)
	if err != nil {
		log.Error().Msg("Bad request")
		return
	}

	var reqData map[string]interface{}
	err = json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		log.Error().Msg("Bad request")
		return
	}

	var data RequestBody

	if json.Unmarshal([]byte(reqData["Body"].(string)), &data) != nil {
		log.Error().Msg("Bad request")
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
		Cluster: WekaClusterParams{
			VmName:               data.Vm,
			HostsNum:             hostsNum,
			Name:                 clusterName,
			ComputeMemory:        computeMemory,
			DrivesContainerNum:   drivesContainerNum,
			ComputeContainerNum:  computeContainerNum,
			FrontendContainerNum: frontendContainerNum,
			TieringSsdPercent:    tieringSsdPercent,
			DataProtection: DataProtectionParams{
				StripeWidth:     stripeWidth,
				ProtectionLevel: protectionLevel,
				Hotspare:        hotspare,
			},
		},
		Obs: ObsParams{
			SetObs:        setObs,
			Name:          obsName,
			ContainerName: obsContainerName,
			AccessKey:     obsAccessKey,
		},
	}

	if data.Vm == "" {
		msg := "Cluster name wasn't supplied"
		log.Error().Msgf(msg)
		resData["body"] = msg
	} else {
		clusterizeScript := Clusterize(params)
		resData["body"] = clusterizeScript
	}
	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
