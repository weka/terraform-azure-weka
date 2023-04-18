package clusterize

import (
	"fmt"
	"strings"
	fd "weka-deployment/lib/functions_def"

	"github.com/lithammer/dedent"
)

type DataProtectionParams struct {
	StripeWidth     int
	ProtectionLevel int
	Hotspare        int
}

type ClusterParams struct {
	VMNames        []string
	IPs            []string
	ClusterName    string
	HostsNum       int
	NvmesNum       int
	WekaUsername   string
	WekaPassword   string
	SetObs         bool
	ObsScript      string
	DataProtection DataProtectionParams
	InstallDpdk    bool
}

type ClusterizeScriptGenerator struct {
	Params  ClusterParams
	FuncDef fd.FunctionDef
}

func (c *ClusterizeScriptGenerator) GetClusterizeScript() string {
	reportFuncDef := c.FuncDef.GetFunctionCmdDefinition(fd.Report)
	clusterizeFinFuncDef := c.FuncDef.GetFunctionCmdDefinition(fd.ClusterizeFinalizaition)
	params := c.Params

	clusterizeScriptTemplate := `
	#!/bin/bash
	set -ex
	VMS=(%s)
	IPS=(%s)
	CLUSTER_NAME=%s
	HOSTS_NUM=%d
	NVMES_NUM=%d
	SET_OBS=%t
	STRIPE_WIDTH=%d
	PROTECTION_LEVEL=%d
	HOTSPARE=%d
	WEKA_USERNAME="%s"
	WEKA_PASSWORD="%s"
	INSTALL_DPDK=%t

	CONTAINER_NAMES=(drives0 compute0 frontend0)
	PORTS=(14000 15000 16000)

	# report function definition
	%s

	# clusterize_finalization function definition
	%s

	HOST_IPS=()
	HOST_NAMES=()
	for i in "${!IPS[@]}"; do
		for j in "${!PORTS[@]}"; do
			HOST_IPS+=($(echo "${IPS[i]}:${PORTS[j]}"))
			HOST_NAMES+=($(echo "${VMS[i]}-${CONTAINER_NAMES[j]}"))
		done
	done
	host_ips=$(IFS=, ;echo "${HOST_IPS[*]}")
	host_names=$(IFS=' ' ;echo "${HOST_NAMES[*]}")

	report "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Running Clusterization\"}"

	vms_string=$(printf "%%s "  "${VMS[@]}" | rev | cut -c2- | rev)
	weka cluster create $host_names --host-ips $host_ips --admin-password "$WEKA_PASSWORD"
	weka user login $WEKA_USERNAME $WEKA_PASSWORD
	
	if [[ $INSTALL_DPDK == true ]]; then
		weka debug override add --key allow_uncomputed_backend_checksum
		weka debug override add --key allow_azure_auto_detection
	fi
	
	sleep 30s

	DRIVE_NUMS=( $(weka cluster container | grep drives | awk '{print $1;}') )
	
	for drive_num in "${DRIVE_NUMS[@]}"; do
		for (( d=0; d<$NVMES_NUM; d++ )); do
			weka cluster drive add $drive_num "/dev/nvme$d"n1
		done
	done

	weka cluster update --cluster-name="$CLUSTER_NAME"

	weka cloud enable
	weka cluster update --data-drives $STRIPE_WIDTH --parity-drives $PROTECTION_LEVEL
	weka cluster hot-spare $HOTSPARE
	weka cluster start-io
	
	sleep 15s
	
	weka cluster process
	weka cluster drive
	weka cluster container
	
	full_capacity=$(weka status -J | jq .capacity.unprovisioned_bytes)
	weka fs group create default
	weka fs create default default "$full_capacity"B
	
	if [[ $SET_OBS == true ]]; then
	  # 'set obs' script
	  %s
	fi

	if [[ $INSTALL_DPDK == true ]]; then
		weka alerts mute NodeRDMANotActive 365d
	else
		weka alerts mute JumboConnectivity 365d
		weka alerts mute UdpModePerformanceWarning 365d
	fi

	echo "completed successfully" > /tmp/weka_clusterization_completion_validation
	report "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Clusterization completed successfully\"}"

	clusterize_finalization "{}"
	`
	script := fmt.Sprintf(
		dedent.Dedent(clusterizeScriptTemplate), strings.Join(params.VMNames, " "), strings.Join(params.IPs, " "), params.ClusterName, params.HostsNum, params.NvmesNum,
		params.SetObs, params.DataProtection.StripeWidth, params.DataProtection.ProtectionLevel, params.DataProtection.Hotspare,
		params.WekaUsername, params.WekaPassword, params.InstallDpdk, reportFuncDef, clusterizeFinFuncDef, params.ObsScript,
	)
	return script
}
