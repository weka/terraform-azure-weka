package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/weka/go-cloud-lib/logging"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
	"weka-deployment/common"

	"github.com/lithammer/dedent"
)

type BackendCoreCount struct {
	total     int
	frontend  int
	drive     int
	converged bool
	memory    int
}

type BackendCoreCounts map[string]BackendCoreCount

func shuffleSlice(slice []string) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(slice), func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func getBackendCoreCountsDefaults() BackendCoreCounts {
	backendCoreCounts := BackendCoreCounts{
		"Standard_L8s_v3":  BackendCoreCount{total: 4, frontend: 1, drive: 1, memory: 31},
		"Standard_L16s_v3": BackendCoreCount{total: 8, frontend: 1, drive: 2, memory: 72},
		"Standard_L32s_v3": BackendCoreCount{total: 8, frontend: 1, drive: 2, memory: 189},
		"Standard_L48s_v3": BackendCoreCount{total: 8, frontend: 1, drive: 3, memory: 306},
		"Standard_L64s_v3": BackendCoreCount{total: 8, frontend: 1, drive: 2, memory: 418},
	}
	return backendCoreCounts
}

func getWekaIoToken(ctx context.Context, keyVaultUri string) (token string, err error) {
	token, err = common.GetKeyVaultValue(ctx, keyVaultUri, "get-weka-io-token")
	return
}

func getFunctionKey(ctx context.Context, keyVaultUri string) (functionAppKey string, err error) {
	functionAppKey, err = common.GetKeyVaultValue(ctx, keyVaultUri, "function-app-default-key")
	return
}

func GetJoinParams(ctx context.Context, subscriptionId, resourceGroupName, prefix, clusterName, instanceType, keyVaultUri, functionKey, vm, installDpdk, subnetsRange, nicsNum string) (bashScript string, err error) {
	logger := logging.LoggerFromCtx(ctx)

	joinFinalizationUrl := fmt.Sprintf("https://%s-%s-function-app.azurewebsites.net/api/join_finalization", prefix, clusterName)
	reportUrl := fmt.Sprintf("https://%s-%s-function-app.azurewebsites.net/api/report", prefix, clusterName)

	vmScaleSetName := fmt.Sprintf("%s-%s-vmss", prefix, clusterName)
	vmsPrivateIps, err := common.GetVmsPrivateIps(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		return
	}

	computeName := strings.Split(vm, ":")[0]
	privateIp := vmsPrivateIps[computeName]
	hashedPrivateIp := common.GetHashedPrivateIp(privateIp)

	var ips []string
	for _, ip := range vmsPrivateIps {
		ips = append(ips, ip)
	}

	if len(ips) == 0 {
		err = fmt.Errorf("no instances found for instance group %s, can't join", vmScaleSetName)
		return
	}
	shuffleSlice(ips)
	wekaPassword, err := common.GetWekaClusterPassword(ctx, keyVaultUri)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	bashScriptTemplate := `
	#!/bin/bash

	set -ex

	export WEKA_USERNAME="admin"
	export WEKA_PASSWORD="%s"
	export WEKA_RUN_CREDS="-e WEKA_USERNAME=$WEKA_USERNAME -e WEKA_PASSWORD=$WEKA_PASSWORD"
	declare -a backend_ips=("%s" )
	FUNCTION_KEY="%s"
	REPORT_URL="%s"
	HASHED_IP="%s"

	curl $REPORT_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Joining started\"}"

	random=$$
	echo $random
	for backend_ip in ${backend_ips[@]}; do
		if VERSION=$(curl -s -XPOST --data '{"jsonrpc":"2.0", "method":"client_query_backend", "id":"'$random'"}' $backend_ip:14000/api/v1 | sed  's/.*"software_release":"\([^"]*\)".*$/\1/g'); then
			if [[ "$VERSION" != "" ]]; then
				break
			fi
		fi
	done

	ip=$(ifconfig eth0 | grep "inet " | awk '{ print $2}')
	while [ ! $ip ] ; do
		sleep 1
		ip=$(ifconfig eth0 | grep "inet " | awk '{ print $2}')
	done

	COMPUTE=%d
	FRONTEND=%d
	DRIVES=%d
	COMPUTE_MEMORY=%d
	INSTALL_DPDK=%s
	SUBNETS_RANGE="%s"
	NICS_NUM=%s
	IPS=%s

	function getNetStrForDpdk() {
		i=$1
		j=$2
		net=" "
		gateway=$(route -n | grep 0.0.0.0 | grep UG | awk '{print $2}')
		for ((i; i<$j; i++)); do
			eth=$(ifconfig | grep eth$i -C2 | grep 'inet ' | awk '{print $2}')
			enp=$(ls -l /sys/class/net/eth$i/ | grep lower | awk -F"_" '{print $2}' | awk '{print $1}')
			bits=$(ip -o -f inet addr show eth$i | awk '{print $4}')
			IFS='/' read -ra netmask <<< "$bits"
			net="$net --net $enp/$eth/${netmask[1]}/$gateway"
		done
	}

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
	get_core_ids $DRIVES drive_core_ids
	get_core_ids $COMPUTE compute_core_ids
	get_core_ids $FRONTEND frontend_core_ids

	if [[ $INSTALL_DPDK == true ]]; then
		for(( i=0; i<$NICS_NUM; i++)); do
				   echo "20$i eth$i-rt" >> /etc/iproute2/rt_tables
		done

		echo "network:"> /etc/cloud/cloud.cfg.d/99-disable-network-config.cfg
		echo "  config: disabled" >> /etc/cloud/cloud.cfg.d/99-disable-network-config.cfg
		subnets=($SUBNETS_RANGE)
		gateway=$(ip r | grep default | awk '{print $3}')
		for(( i=0; i<$NICS_NUM; i++)); do
			eth=$(ifconfig | grep eth$i -C2 | grep 'inet ' | awk '{print $2}')
			cat <<-EOF | sed -i "/            set-name: eth$i/r /dev/stdin" /etc/netplan/50-cloud-init.yaml
            mtu: 3900
            routes:
            - to: ${subnets[0]}
              via: $gateway
              metric: 200
              table: 20$i
            - to: 0.0.0.0/0
              via: $gateway
              table: 20$i
            routing-policy:
            - from: $eth/32
              table: 20$i
            - to: $eth/32
              table: 20$i
		EOF
		done
		netplan apply
	fi

	sleep 30

	curl $REPORT_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Installing weka\"}"
	curl $backend_ip:14000/dist/v1/install | sh
	curl $REPORT_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Weka installation completed\"}"

	weka version get --from $backend_ip:14000 $VERSION --set-current
	weka version prepare $VERSION
	weka local stop && weka local rm --all -f

	if [[ $INSTALL_DPDK == true ]]; then
		getNetStrForDpdk 1 $(($DRIVES+1))
		weka local setup container --name drives0 --base-port 14000 --cores $DRIVES --no-frontends --drives-dedicated-cores $DRIVES --join-ips $IPS --failure-domain "$HASHED_IP" --core-ids $drive_core_ids $net --dedicate

		getNetStrForDpdk $((1+$DRIVES)) $((1+$DRIVES+$COMPUTE))
		weka local setup container --name compute0 --base-port 15000 --cores $COMPUTE --memory "$COMPUTE_MEMORY"GB --no-frontends --compute-dedicated-cores $COMPUTE --join-ips $IPS --failure-domain "$HASHED_IP" --core-ids $compute_core_ids $net --dedicate

		getNetStrForDpdk $(($NICS_NUM-1)) $(($NICS_NUM))
		weka local setup container --name frontend0 --base-port 16000 --cores $FRONTEND --allow-protocols true --frontend-dedicated-cores $FRONTEND --join-ips $IPS --failure-domain "$HASHED_IP" --core-ids $frontend_core_ids $net --dedicate
	else
		weka local setup container --name drives0 --base-port 14000 --cores $DRIVES --no-frontends --drives-dedicated-cores $DRIVES --join-ips $IPS --failure-domain "$HASHED_IP" --core-ids $drive_core_ids --dedicate
		weka local setup container --name compute0 --base-port 15000 --cores $COMPUTE --memory "$COMPUTE_MEMORY"GB --no-frontends --compute-dedicated-cores $COMPUTE --join-ips $IPS --failure-domain "$HASHED_IP" --core-ids $compute_core_ids --dedicate
		weka local setup container --name frontend0 --base-port 16000 --cores $FRONTEND --allow-protocols true --frontend-dedicated-cores $FRONTEND --join-ips $IPS --failure-domain "$HASHED_IP" --core-ids $frontend_core_ids --dedicate
	fi`

	isReady := `
	while ! weka debug manhole -s 0 operational_status | grep '"is_ready": true' ; do
		sleep 1
	done
	echo Connected to cluster
	`

	// we will use here the FUNCTION_KEY and REPORT_URL from the bashScriptTemplate
	addDrives := `
	JOIN_FINALIZATION_URL="%s"

	host_id=$(weka local run --container compute0 $WEKA_RUN_CREDS manhole getServerInfo | grep hostIdValue: | awk '{print $2}')
	mkdir -p /opt/weka/tmp
	cat >/opt/weka/tmp/find_drives.py <<EOL
import json
import sys
for d in json.load(sys.stdin)['disks']:
	if d['isRotational']: continue
	print(d['devPath'])
EOL
	devices=$(weka local run --container compute0 $WEKA_RUN_CREDS bash -ce 'wapi machine-query-info --info-types=DISKS -J | python3 /opt/weka/tmp/find_drives.py')
	for device in $devices; do
		weka local exec --container drives0 /weka/tools/weka_sign_drive $device
	done
	ready=0
	while [ $ready -eq 0 ] ; do
		ready=1
		lsblk
		for device in $devices; do
			if [ ! "$(lsblk | grep ${device#"/dev/"} | grep part)" ]; then
				ready=0
				sleep 5
				break
			fi
		done
	done
	weka cluster drive scan $host_id
	compute_name=$(curl -s -H Metadata:true --noproxy "*" "http://169.254.169.254/metadata/instance?api-version=2021-02-01" | jq '.compute.name')
	compute_name=$(echo "$compute_name" | cut -c2- | rev | cut -c2- | rev)
	curl $JOIN_FINALIZATION_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json" -d "{\"name\": \"$compute_name\"}"
	echo "completed successfully" > /tmp/weka_join_completion_validation
	curl $REPORT_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Join completed successfully\"}"
	`
	var compute, frontend, drive, mem int
	backendCoreCounts := getBackendCoreCountsDefaults()
	instanceParams, ok := backendCoreCounts[instanceType]
	if !ok {
		err = fmt.Errorf("unsupported instance type: %s", instanceType)
		return
	}
	frontend = instanceParams.frontend
	drive = instanceParams.drive
	compute = instanceParams.total - frontend - drive - 1
	mem = instanceParams.memory

	bashScriptTemplate += isReady + fmt.Sprintf(addDrives, joinFinalizationUrl)

	bashScript = fmt.Sprintf(bashScriptTemplate, wekaPassword, strings.Join(ips, "\" \""), functionKey, reportUrl, hashedPrivateIp, compute, frontend, drive, mem, installDpdk, subnetsRange, nicsNum, strings.Join(ips, ","))

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
	vm,
	installDpdk,
	subnetsRange,
	nicsNum string) (bashScript string, err error) {

	clusterizeUrl := fmt.Sprintf("https://%s-%s-function-app.azurewebsites.net/api/clusterize", prefix, clusterName)
	reportUrl := fmt.Sprintf("https://%s-%s-function-app.azurewebsites.net/api/report", prefix, clusterName)
	protectUrl := fmt.Sprintf("https://%s-%s-function-app.azurewebsites.net/api/protect", prefix, clusterName)

	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	functionKey, err := getFunctionKey(ctx, keyVaultUri)
	if err != nil {
		return
	}

	if !state.Clusterized {
		if strings.HasSuffix(installUrl, ".tar") {
			split := strings.Split(installUrl, "/")
			tarName := split[len(split)-1]
			packageName := strings.TrimSuffix(tarName, ".tar")
			installTemplate := `
			#!/bin/bash
			set -ex
			INSTALL_URL=%s
			TAR_NAME=%s
			PACKAGE_NAME=%s
			CLUSTERIZE_URL=%s
			REPORT_URL=%s
			PROTECT_URL=%s
			FUNCTION_KEY=%s
			VM=%s
			INSTALL_DPDK=%s
			SUBNETS_RANGE="%s"
			NICS_NUM=%s	

			if [[ $INSTALL_DPDK == true ]]; then
				subnets=($SUBNETS_RANGE)
				for(( i=0; i<$NICS_NUM; i++)); do
						   echo "20$i eth$i-rt" >> /etc/iproute2/rt_tables
				done

				echo "network:"> /etc/cloud/cloud.cfg.d/99-disable-network-config.cfg
				echo "  config: disabled" >> /etc/cloud/cloud.cfg.d/99-disable-network-config.cfg

				gateway=$(ip r | grep default | awk '{print $3}')
				for(( i=0; i<$NICS_NUM; i++)); do
					eth=$(ifconfig | grep eth$i -C2 | grep 'inet ' | awk '{print $2}')
					cat <<-EOF | sed -i "/            set-name: eth$i/r /dev/stdin" /etc/netplan/50-cloud-init.yaml
            mtu: 3900
            routes:
            - to: ${subnets[0]}
              via: $gateway
              metric: 200
              table: 20$i
            - to: 0.0.0.0/0
              via: $gateway
              table: 20$i
            routing-policy:
            - from: $eth/32
              table: 20$i
            - to: $eth/32
              table: 20$i
				EOF
				done
				netplan apply
			fi

			sleep 30

			gsutil cp $INSTALL_URL /tmp
			cd /tmp
			tar -xvf $TAR_NAME
			cd $PACKAGE_NAME
			curl $REPORT_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Installing weka\"}"
			./install.sh
			curl $REPORT_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Weka installation completed\"}"

			weka local stop
			weka local rm default --force

			curl $PROTECT_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json" -d "{\"vm\": \"$VM\"}"
			curl $CLUSTERIZE_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json" -d "{\"vm\": \"$VM\"}" > /tmp/clusterize.sh
			chmod +x /tmp/clusterize.sh
			/tmp/clusterize.sh 2>&1 | tee /tmp/weka_clusterization.log
			`
			bashScript = fmt.Sprintf(installTemplate, installUrl, tarName, packageName, clusterizeUrl, reportUrl, protectUrl, functionKey, vm, installDpdk, subnetsRange, nicsNum)

		} else {
			token, err2 := getWekaIoToken(ctx, keyVaultUri)
			if err2 != nil {
				err = err2
				return
			}

			installTemplate := `
			#!/bin/bash
			set -ex
			TOKEN=%s
			INSTALL_URL=%s
			CLUSTERIZE_URL=%s
			REPORT_URL=%s
			PROTECT_URL=%s
			FUNCTION_KEY=%s
			VM=%s
			INSTALL_DPDK=%s
			SUBNETS_RANGE="%s"
			NICS_NUM=%s

			# https://gist.github.com/fungusakafungus/1026804
			function retry {
					local retry_max=$1
					local retry_sleep=$2
					shift 2
					local count=$retry_max
					while [ $count -gt 0 ]; do
							"$@" && break
							count=$(($count - 1))
							sleep $retry_sleep
					done
					[ $count -eq 0 ] && {
							echo "Retry failed [$retry_max]: $@"
							return 1
					}
					return 0
			}

			if [[ $INSTALL_DPDK == true ]]; then
				subnets=($SUBNETS_RANGE)
				for(( i=0; i<$NICS_NUM; i++)); do
						   echo "20$i eth$i-rt" >> /etc/iproute2/rt_tables
				done

				echo "network:"> /etc/cloud/cloud.cfg.d/99-disable-network-config.cfg
				echo "  config: disabled" >> /etc/cloud/cloud.cfg.d/99-disable-network-config.cfg
				gateway=$(ip r | grep default | awk '{print $3}')
				for(( i=0; i<$NICS_NUM; i++)); do
					eth=$(ifconfig | grep eth$i -C2 | grep 'inet ' | awk '{print $2}')
					cat <<-EOF | sed -i "/            set-name: eth$i/r /dev/stdin" /etc/netplan/50-cloud-init.yaml
            mtu: 3900
            routes:
            - to: ${subnets[0]}
              via: $gateway
              metric: 200
              table: 20$i
            - to: 0.0.0.0/0
              via: $gateway
              table: 20$i
            routing-policy:
            - from: $eth/32
              table: 20$i
            - to: $eth/32
              table: 20$i
				EOF
				done
				netplan apply
			fi

			sleep 30

			curl $REPORT_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Installing weka\"}"
			retry 300 2 curl --fail --max-time 10 $INSTALL_URL | sh
			curl $REPORT_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Weka installation completed\"}"

			weka local stop
			weka local rm default --force

			curl $PROTECT_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json" -d "{\"vm\": \"$VM\"}"
			curl $CLUSTERIZE_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json" -d "{\"vm\": \"$VM\"}" > /tmp/clusterize.sh
			chmod +x /tmp/clusterize.sh
			/tmp/clusterize.sh > /tmp/cluster_creation.log 2>&1
			`
			bashScript = fmt.Sprintf(installTemplate, token, installUrl, clusterizeUrl, reportUrl, protectUrl, functionKey, vm, installDpdk, subnetsRange, nicsNum)
		}
	} else {
		bashScript, err = GetJoinParams(ctx, subscriptionId, resourceGroupName, prefix, clusterName, instanceType, keyVaultUri, functionKey, vm, installDpdk, subnetsRange, nicsNum)
		if err != nil {
			return
		}
	}

	bashScript = dedent.Dedent(bashScript)
	return
}

type RequestBody struct {
	Vm string `json:"vm"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	clusterName := os.Getenv("CLUSTER_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")

	instanceType := os.Getenv("INSTANCE_TYPE")
	installUrl := os.Getenv("INSTALL_URL")
	installDpdk := os.Getenv("INSTALL_DPDK")
	subnetsRange := os.Getenv("SUBNETS_RANGE")
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
		installDpdk,
		subnetsRange,
		nicsNum)

	if err != nil {
		resData["body"] = err.Error()
	} else {
		resData["body"] = bashScript
	}
	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
