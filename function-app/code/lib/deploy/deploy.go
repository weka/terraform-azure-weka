package deploy

import (
	"fmt"
	"strings"

	fd "weka-deployment/lib/functions_def"

	"github.com/lithammer/dedent"
)

type DeploymentParams struct {
	VMName               string
	ComputeMemory        string
	ComputeContainerNum  string
	FrontendContainerNum string
	DriveContainerNum    string
	WekaInstallUrl       string
	WekaToken            string
}

type DeployScriptGenerator struct {
	FailureDomainCmd string
	Params           DeploymentParams
	FuncDef          fd.FunctionDef
}

func (d *DeployScriptGenerator) GetDeployScript() string {
	wekaInstallScript := d.GetWekaInstallScript()
	protectFunc := d.FuncDef.GetFunctionCmdDefinition(fd.Protect)
	clusterizeFunc := d.FuncDef.GetFunctionCmdDefinition(fd.Clusterize)

	template := `
	#!/bin/bash
	set -ex
	VM=%s
	FAILURE_DOMAIN=$(%s)
	COMPUTE_MEMORY=%s
	NUM_COMPUTE_CONTAINERS=%s
	NUM_FRONTEND_CONTAINERS=%s
	NUM_DRIVE_CONTAINERS=%s

	# clusterize function definition
	%s

	# protect function definition (if any)
	%s

	# install script
	%s

	weka local stop
	weka local rm default --force

	# weka containers setup
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

	sudo weka local setup container --name drives0 --base-port 14000 --cores $NUM_DRIVE_CONTAINERS --no-frontends --drives-dedicated-cores $NUM_DRIVE_CONTAINERS --failure-domain $FAILURE_DOMAIN --core-ids $drive_core_ids
	sudo weka local setup container --name compute0 --base-port 15000 --cores $NUM_COMPUTE_CONTAINERS --no-frontends --compute-dedicated-cores $NUM_COMPUTE_CONTAINERS  --memory $COMPUTE_MEMORY --failure-domain $FAILURE_DOMAIN --core-ids $compute_core_ids
	sudo weka local setup container --name frontend0 --base-port 16000 --cores $NUM_FRONTEND_CONTAINERS --frontend-dedicated-cores $NUM_FRONTEND_CONTAINERS --allow-protocols true --failure-domain $FAILURE_DOMAIN --core-ids $frontend_core_ids

	# should not call 'clusterize' untill all 3 containers are up
	ready_containers=0
	while [ $ready_containers -ne 3 ];
	do
		sleep 10
		ready_containers=$( weka local ps | grep -i 'running' | wc -l )
		echo "Running containers: $ready_containers"
	done

	protect "{\"vm\": \"$VM\"}"
	clusterize "{\"vm\": \"$VM\"}" > /tmp/clusterize.sh
	chmod +x /tmp/clusterize.sh
	/tmp/clusterize.sh 2>&1 | tee /tmp/weka_clusterization.log
	`
	return fmt.Sprintf(
		template, d.Params.VMName, d.FailureDomainCmd, d.Params.ComputeMemory, d.Params.ComputeContainerNum,
		d.Params.FrontendContainerNum, d.Params.DriveContainerNum, clusterizeFunc, protectFunc, wekaInstallScript,
	)
}

func (d *DeployScriptGenerator) GetWekaInstallScript() string {
	installUrl := d.Params.WekaInstallUrl
	reportFuncDef := d.FuncDef.GetFunctionCmdDefinition(fd.Report)

	installScriptTemplate := `
	# report function definition
	%s
	TOKEN=%s
	INSTALL_URL=%s
	`
	installScript := fmt.Sprintf(installScriptTemplate, reportFuncDef, d.Params.WekaToken, installUrl)

	if strings.HasSuffix(installUrl, ".tar") {
		split := strings.Split(installUrl, "/")
		tarName := split[len(split)-1]
		packageName := strings.TrimSuffix(tarName, ".tar")
		installTemplate := `
		TAR_NAME=%s
		PACKAGE_NAME=%s

		gsutil cp $INSTALL_URL /tmp
		cd /tmp
		tar -xvf $TAR_NAME
		cd $PACKAGE_NAME

		report "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Installing weka\"}"
		./install.sh
		report "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Weka installation completed\"}"
		`
		installScript += fmt.Sprintf(installTemplate, tarName, packageName)
	} else {
		installScript += `
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

		report "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Installing weka\"}"
		retry 300 2 curl --fail --max-time 10 $INSTALL_URL | sh
		report "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Weka installation completed\"}"
		`
	}
	return dedent.Dedent(installScript)
}
