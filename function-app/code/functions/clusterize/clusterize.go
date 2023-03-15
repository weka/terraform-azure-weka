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

	"github.com/weka/go-cloud-lib/logging"

	"github.com/lithammer/dedent"
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
	NvmesNum             string
	ComputeContainerNum  string
	FrontendContainerNum string
	driveContainerNum    string
	TieringSsdPercent    string
	DataProtection       DataProtectionParams
	InstallDpdk          string
	NicsNum              string
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
	ctx context.Context, vmNames, ips, prefix, functionAppKey, wekaPassword string, cluster WekaClusterParams, obs ObsParams, hashedIps []string,
) (clusterizeScript string) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("Generating clusterization script")

	clusterizeScriptTemplate := `
	#!/bin/bash
	
	set -ex
	VMS=(%s)
	IPS=%s
	HOSTS_NUM=%s
	NVMES_NUM=%s
	CLUSTER_NAME=%s
	NUM_COMPUTE_CONTAINERS=%s
	COMPUTE_MEMORY=%s
	NUM_FRONTEND_CONTAINERS=%s
	NUM_DRIVE_CONTAINERS=%s
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
	INSTALL_DPDK=%s
	NICS_NUM=%s
	HASHED_IPS=(%s)

	report_url=https://$PREFIX-$CLUSTER_NAME-function-app.azurewebsites.net/api/report
	clusterize_finalization_url=https://$PREFIX-$CLUSTER_NAME-function-app.azurewebsites.net/api/clusterize_finalization

	curl "$report_url?code=$FUNCTION_APP_KEY" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Running Clusterization\"}"

	ssh_command="ssh -o StrictHostKeyChecking=no"

	core_ids=$(cat /sys/devices/system/cpu/cpu*/topology/thread_siblings_list | cut -d "-" -f 1 | sort -u | tr '\n' ' ')
	core_ids="${core_ids[@]/0}"
	IFS=', ' read -r -a core_ids <<< "$core_ids"
	core_idx_begin=0
	core_idx_end=$(($core_idx_begin + $NUM_DRIVE_CONTAINERS))
	get_core_ids() {
		core_idx_end=$(($core_idx_begin + $1))
		res=${core_ids[i]}
		for (( i=$(($core_idx_begin + 1)); i<$core_idx_end; i++ ))
		do
			res=$res,${core_ids[i]}
		done
		core_idx_begin=$core_idx_end
		eval "$2=$res"
	}
	get_core_ids $NUM_DRIVE_CONTAINERS drive_core_ids
	get_core_ids $NUM_COMPUTE_CONTAINERS compute_core_ids
	get_core_ids $NUM_FRONTEND_CONTAINERS frontend_core_ids

	for index in ${!VMS[*]}; do
		hashed_ip=${HASHED_IPS[$index]}
		$ssh_command ${VMS[$index]} "sudo weka local setup container --name drives0 --base-port 14000 --cores $NUM_DRIVE_CONTAINERS --no-frontends --drives-dedicated-cores $NUM_DRIVE_CONTAINERS --failure-domain $hashed_ip --core-ids $drive_core_ids"
	done
	
	vms_string=$(printf "%%s "  "${VMS[@]}" | rev | cut -c2- | rev)
	weka cluster create $vms_string --host-ips $IPS --admin-password "$WEKA_PASSWORD"
	weka user login admin "$WEKA_PASSWORD"
	if [[ $INSTALL_DPDK == true ]]; then
		weka debug override add --key allow_azure_auto_detection
		weka debug override add --key allow_uncomputed_backend_checksum
	fi
	
	sleep 30s
	
	for (( i=0; i<$HOSTS_NUM; i++ )); do
		for (( d=0; d<$NVMES_NUM; d++ )); do
			weka cluster drive add $i "/dev/nvme$d"n1
		done
	done

	weka cluster update --cluster-name="$CLUSTER_NAME"
	
	for index in ${!VMS[*]}; do
		hashed_ip=${HASHED_IPS[$index]}
		net=""
      	if [[ $INSTALL_DPDK == true ]]; then
			i=$(($NICS_NUM-1-$NUM_COMPUTE_CONTAINERS))
			j=$(($i+$NUM_COMPUTE_CONTAINERS))
			net=""
			for ((i; i<$j; i++)); do
				enp=$($ssh_command $vm "ls -l /sys/class/net/eth$i/ | grep enP" | awk -F"_" '{print $2}' | awk '{print $1}')
				eth=$($ssh_command $vm "ifconfig | grep eth$i -C2 | grep 'inet '" | awk '{print $2}')
				gateway=$(echo ${eth::-1}1)
				bits=$(ip -o -f inet addr show eth$i | awk '{print $4}')
            	IFS='/' read -ra netmask <<< "$bits"
				net="$net --net $enp/$eth/${netmask[1]}/$gateway "
			done
			$ssh_command ${VMS[$index]} "sudo weka local setup container --name compute0 --base-port 15000 --cores $NUM_COMPUTE_CONTAINERS --no-frontends --compute-dedicated-cores $NUM_COMPUTE_CONTAINERS  --memory $COMPUTE_MEMORY --join-ips $IPS --failure-domain $hashed_ip $net --core-ids $compute_core_ids"
		else
			$ssh_command ${VMS[$index]} "sudo weka local setup container --name frontend0 --base-port 16000 --cores $NUM_FRONTEND_CONTAINERS --frontend-dedicated-cores $NUM_FRONTEND_CONTAINERS --allow-protocols true --join-ips $IPS --failure-domain $hashed_ip --core-ids $compute_core_ids"
		fi
	done
	
	weka cloud enable
	weka cluster update --data-drives $STRIPE_WIDTH --parity-drives $PROTECTION_LEVEL
	weka cluster hot-spare $HOTSPARE
	weka cluster start-io
	
	for index in ${!VMS[*]}; do
		hashed_ip=${HASHED_IPS[$index]}
		net=""
		if [[ $INSTALL_DPDK == true ]]; then
			i=$(($NICS_NUM-1))
			enp=$($ssh_command $vm "ls -l /sys/class/net/eth$i/ | grep enP" | awk -F"_" '{print $2}' | awk '{print $1}')
			eth=$($ssh_command $vm "ifconfig | grep eth$i -C2 | grep 'inet '" | awk '{print $2}')
			gateway=$(echo ${eth::-1}1)
			bits=$(ip -o -f inet addr show eth$i | awk '{print $4}')
            IFS='/' read -ra netmask <<< "$bits"
			net="$net --net $enp/$eth/${netmask[1]}/$gateway "
			$ssh_command ${VMS[$index]} "sudo weka local setup container --name drives0 --base-port 14000 --cores $NUM_DRIVE_CONTAINERS --no-frontends --drives-dedicated-cores $NUM_DRIVE_CONTAINERS --failure-domain $hashed_ip $net --core-ids $frontend_core_ids"

		else
			$ssh_command ${VMS[$index]} "sudo weka local setup container --name drives0 --base-port 14000 --cores $NUM_DRIVE_CONTAINERS --no-frontends --drives-dedicated-cores $NUM_DRIVE_CONTAINERS --failure-domain $hashed_ip --core-ids $frontend_core_ids"
		fi
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
	
	if [[ $INSTALL_DPDK == true ]]; then
		weka alerts mute JumboConnectivity 365d
		weka alerts mute UdpModePerformanceWarning 365d
	fi

	echo "completed successfully" > /tmp/weka_clusterization_completion_validation
	curl "$report_url?code=$FUNCTION_APP_KEY" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Clusterization completed successfully\"}"

	curl "$clusterize_finalization_url?code=$FUNCTION_APP_KEY"
	`

	logger.Info().Msgf("Formatting clusterization script template")
	clusterizeScript = fmt.Sprintf(
		dedent.Dedent(clusterizeScriptTemplate), vmNames, ips, cluster.HostsNum, cluster.NvmesNum, cluster.Name,
		cluster.ComputeContainerNum, cluster.ComputeMemory, cluster.FrontendContainerNum, cluster.driveContainerNum,
		obs.SetObs, obs.Name, obs.ContainerName, obs.AccessKey, cluster.TieringSsdPercent, prefix, functionAppKey,
		cluster.DataProtection.StripeWidth, cluster.DataProtection.ProtectionLevel, cluster.DataProtection.Hotspare,
		wekaPassword, cluster.InstallDpdk, cluster.NicsNum, strings.Join(hashedIps, " "),
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

func reportClusterizeError(ctx context.Context, p ClusterizationParams, err error) {
	hostName := strings.Split(p.Cluster.VmName, ":")[1]
	report := common.Report{Type: "error", Hostname: hostName, Message: err.Error()}
	_ = common.UpdateStateReporting(ctx, p.SubscriptionId, p.ResourceGroupName, p.StateContainerName, p.StateStorageName, report)
}

func HandleLastClusterVm(ctx context.Context, state common.ClusterState, p ClusterizationParams) (clusterizeScript string) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("This is the last instance in the cluster, creating obs and clusterization script")

	var err error
	vmScaleSetName := common.GetVmScaleSetName(p.Prefix, p.Cluster.Name)

	if p.Obs.SetObs == "true" {
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
	vmNames := strings.Join(vmNamesList, " ")
	ips := strings.Join(ipsList, ",")

	var hashedIps []string
	for _, privateIp := range ipsList {
		hashedIps = append(hashedIps, common.GetHashedPrivateIp(privateIp))
	}

	clusterizeScript = generateClusterizationScript(ctx, vmNames, ips, p.Prefix, functionAppKey, wekaPassword, p.Cluster, p.Obs, hashedIps)
	return
}

func Clusterize(ctx context.Context, p ClusterizationParams) (clusterizeScript string) {
	logger := logging.LoggerFromCtx(ctx)

	instanceName := strings.Split(p.Cluster.VmName, ":")[0]
	instanceId := common.GetScaleSetVmIndex(instanceName)
	vmScaleSetName := common.GetVmScaleSetName(p.Prefix, p.Cluster.Name)

	ip, err := common.GetPublicIp(ctx, p.SubscriptionId, p.ResourceGroupName, vmScaleSetName, p.Prefix, p.Cluster.Name, instanceId)

	vmName := p.Cluster.VmName
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

	initialSize, err := strconv.Atoi(p.Cluster.HostsNum)
	if err != nil {
		clusterizeScript = GetErrorScript(err)
		return
	}

	if len(state.Instances) == initialSize {
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
	nvmesNum := os.Getenv("NVMES_NUM")
	computeContainerNum := os.Getenv("NUM_COMPUTE_CONTAINERS")
	frontendContainerNum := os.Getenv("NUM_FRONTEND_CONTAINERS")
	driveContainerNum := os.Getenv("NUM_DRIVE_CONTAINERS")
	tieringSsdPercent := os.Getenv("TIERING_SSD_PERCENT")
	prefix := os.Getenv("PREFIX")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")
	// data protection-related vars
	stripeWidth, _ := strconv.Atoi(os.Getenv("STRIPE_WIDTH"))
	protectionLevel, _ := strconv.Atoi(os.Getenv("PROTECTION_LEVEL"))
	hotspare, _ := strconv.Atoi(os.Getenv("HOTSPARE"))
	installDpdk := os.Getenv("INSTALL_DPDK")
	nicsNum := os.Getenv("NICS_NUM")

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
		Cluster: WekaClusterParams{
			VmName:               data.Vm,
			HostsNum:             hostsNum,
			Name:                 clusterName,
			ComputeMemory:        computeMemory,
			NvmesNum:             nvmesNum,
			ComputeContainerNum:  computeContainerNum,
			FrontendContainerNum: frontendContainerNum,
			driveContainerNum:    driveContainerNum,
			TieringSsdPercent:    tieringSsdPercent,
			InstallDpdk:          installDpdk,
			NicsNum:              nicsNum,
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
