package deploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lithammer/dedent"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
	"weka-deployment/common"
)

type BackendCoreCount struct {
	total     int
	frontend  int
	drive     int
	converged bool
}

type BackendCoreCounts map[string]BackendCoreCount

func shuffleSlice(slice []string) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(slice), func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func getBackendCoreCountsDefaults() BackendCoreCounts {
	backendCoreCounts := BackendCoreCounts{
		"Standard_L8s_v3":  BackendCoreCount{total: 8, frontend: 1, drive: 1},
		"Standard_L16s_v3": BackendCoreCount{total: 4, frontend: 1, drive: 2},
	}
	return backendCoreCounts
}

func getWekaIoToken(keyVaultUri string) (token string, err error) {
	token, err = common.GetKeyVaultValue(keyVaultUri, "get-weka-io-token")
	return
}

func getFunctionKey(keyVaultUri string) (functionAppKey string, err error) {
	functionAppKey, err = common.GetKeyVaultValue(keyVaultUri, "function-app-default-key")
	return
}

func GetJoinParams(subscriptionId, resourceGroupName, prefix, clusterName, instanceType, subnet string) (bashScript string, err error) {
	vmScaleSetName := fmt.Sprintf("%s-%s-vmss", prefix, clusterName)
	vmsPrivateIps, err := common.GetVmsPrivateIps(subscriptionId, resourceGroupName, vmScaleSetName)

	var ips []string
	for _, ip := range vmsPrivateIps {
		ips = append(ips, ip)
	}

	if len(ips) == 0 {
		err = errors.New(fmt.Sprintf("No instances found for instance group %s, can't join", vmScaleSetName))
		return
	}
	shuffleSlice(ips)
	//creds, err := common.GetUsernameAndPassword(usernameId, passwordId)
	//if err != nil {
	//	log.Error().Msgf("%s", err)
	//	return
	//}

	bashScriptTemplate := `
	#!/bin/bash

	set -ex

	#export WEKA_USERNAME="%s"
	#export WEKA_PASSWORD="%s"
	#export WEKA_RUN_CREDS="-e WEKA_USERNAME=$WEKA_USERNAME -e WEKA_PASSWORD=$WEKA_PASSWORD"
	declare -a backend_ips=("%s" )

	random=$$
	echo $random
	for backend_ip in ${backend_ips[@]}; do
		if VERSION=$(curl -s -XPOST --data '{"jsonrpc":"2.0", "method":"client_query_backend", "id":"'$random'"}' $backend_ip:14000/api/v1 | sed  's/.*"software_release":"\([^"]*\)".*$/\1/g'); then
			if [[ "$VERSION" != "" ]]; then
				break
			fi
		fi
	done

	SUBNET=%s

	ip=$(ifconfig eth$i | grep "inet " | awk '{ print $2}')
	while [ ! $ip ] ; do
		sleep 1
		ip=$(ifconfig eth$i | grep "inet " | awk '{ print $2}')
	done

	curl $backend_ip:14000/dist/v1/install | sh

	weka version get --from $backend_ip:14000 $VERSION --set-current
	weka version prepare $VERSION
	weka local stop && weka local rm --all -f

	weka local setup host --cores %d --frontend-dedicated-cores %d --drives-dedicated-cores %d --join-ips %s --net eth$i/$ip`

	isReady := `
	while ! weka debug manhole -s 0 operational_status | grep '"is_ready": true' ; do
		sleep 1
	done
	echo Connected to cluster
	`

	addDrives := `
	#JOIN_FINALIZATION_URL=%s
	host_id=$(weka local run $WEKA_RUN_CREDS manhole getServerInfo | grep hostIdValue: | awk '{print $2}')
	mkdir -p /opt/weka/tmp
	cat >/opt/weka/tmp/find_drives.py <<EOL
	import json
	import sys
	for d in json.load(sys.stdin)['disks']:
		if d['isRotational']: continue
		if d['type'] != 'DISK': continue
		if d['isMounted']: continue
		if d['model'] != 'nvme_card': continue
		print(d['devPath'])
	EOL
	devices=$(weka local run $WEKA_RUN_CREDS bash -ce 'wapi machine-query-info --info-types=DISKS -J | python3 /opt/weka/tmp/find_drives.py')
	for device in $devices; do
		weka local exec /weka/tools/weka_sign_drive $device
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
	#curl $JOIN_FINALIZATION_URL -H "Content-Type:application/json"  -d "{\"name\": \"$HOSTNAME\"}"
	echo "completed successfully" > /tmp/weka_join_completion_validation
	`
	var cores, frontend, drive int
	backendCoreCounts := getBackendCoreCountsDefaults()
	instanceParams, ok := backendCoreCounts[instanceType]
	if !ok {
		err = errors.New(fmt.Sprintf("Unsupported instance type: %s", instanceType))
		return
	}
	cores = 1
	frontend = instanceParams.frontend
	drive = instanceParams.drive
	if !instanceParams.converged {
		bashScriptTemplate += " --dedicate"
	}
	bashScriptTemplate += isReady + addDrives

	bashScript = fmt.Sprintf(bashScriptTemplate, strings.Join(ips, "\" \""), subnet, cores, frontend, drive, strings.Join(ips, ","))

	return
}

func GetDeployScript(
	subscriptionId,
	resourceGroupName,
	stateStorageName,
	stateContainerName,
	prefix,
	clusterName,
	instanceType,
	installUrl,
	keyVaultUri,
	clusterizeUrl,
	subnet string) (bashScript string, err error) {

	state, err := common.ReadState(stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	functionKey, err := getFunctionKey(keyVaultUri)
	if err != nil {
		return
	}

	backendCoreCounts := getBackendCoreCountsDefaults()
	instanceParams, ok := backendCoreCounts[instanceType]
	if !ok {
		err = errors.New(fmt.Sprintf("Unsupported instance type: %s", instanceType))
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
			FUNCTION_KEY=%s
			DRIVE_CONTAINERS_NUM=%d

			gsutil cp $INSTALL_URL /tmp
			cd /tmp
			tar -xvf $TAR_NAME
			cd $PACKAGE_NAME
			./install.sh

			weka local stop
			weka local rm default --force
			weka local setup container --name drives0 --base-port 14000 --cores $DRIVE_CONTAINERS_NUM --no-frontends --drives-dedicated-cores $DRIVE_CONTAINERS_NUM

			compute_name=$(curl -s -H Metadata:true --noproxy "*" "http://169.254.169.254/metadata/instance?api-version=2021-02-01" | jq '.compute.name')
			compute_name=$(echo "$compute_name" | cut -c2- | rev | cut -c2- | rev)
			curl $CLUSTERIZE_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json"  -d "{\"vm\": \"$compute_name:$HOSTNAME\"}" > /tmp/clusterize.sh
			chmod +x /tmp/clusterize.sh
			/tmp/clusterize.sh > /tmp/cluster_creation.log 2>&1
			`
			bashScript = fmt.Sprintf(installTemplate, installUrl, tarName, packageName, clusterizeUrl, functionKey, instanceParams.drive)

		} else {
			token, err2 := getWekaIoToken(keyVaultUri)
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
			FUNCTION_KEY=%s
			DRIVE_CONTAINERS_NUM=%d

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

			retry 300 2 curl --fail --max-time 10 $INSTALL_URL | sh

			weka local stop
			weka local rm default --force
			weka local setup container --name drives0 --base-port 14000 --cores $DRIVE_CONTAINERS_NUM --no-frontends --drives-dedicated-cores $DRIVE_CONTAINERS_NUM

			compute_name=$(curl -s -H Metadata:true --noproxy "*" "http://169.254.169.254/metadata/instance?api-version=2021-02-01" | jq '.compute.name')
			compute_name=$(echo "$compute_name" | cut -c2- | rev | cut -c2- | rev)
			curl $CLUSTERIZE_URL?code="$FUNCTION_KEY" -H "Content-Type:application/json"  -d "{\"vm\": \"$compute_name:$HOSTNAME\"}" > /tmp/clusterize.sh
			chmod +x /tmp/clusterize.sh
			/tmp/clusterize.sh > /tmp/cluster_creation.log 2>&1
			`
			bashScript = fmt.Sprintf(installTemplate, token, installUrl, clusterizeUrl, functionKey, instanceParams.drive)
		}
	} else {
		bashScript, err = GetJoinParams(subscriptionId, resourceGroupName, prefix, clusterName, instanceType, subnet)
		if err != nil {
			return
		}
	}

	bashScript = dedent.Dedent(bashScript)
	return
}

func Handler(w http.ResponseWriter, r *http.Request) {
	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})

	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	clusterName := os.Getenv("CLUSTER_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")

	subnet := os.Getenv("SUBNET")
	instanceType := os.Getenv("INSTANCE_TYPE")
	installUrl := os.Getenv("INSTALL_URL")
	clusterizeUrl := fmt.Sprintf("https://%s-%s-function-app.azurewebsites.net/api/clusterize", prefix, clusterName)

	bashScript, err := GetDeployScript(
		subscriptionId,
		resourceGroupName,
		stateStorageName,
		stateContainerName,
		prefix,
		clusterName,
		instanceType,
		installUrl,
		keyVaultUri,
		clusterizeUrl,
		subnet)

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
