FAILURE_DOMAIN=$(printf $(hostname -I) | sha256sum | tr -d '-' | cut -c1-16)
FRONTEND_CONTAINER_CORES_NUM=${frontend_container_cores_num}
NICS_NUM=$((FRONTEND_CONTAINER_CORES_NUM+1))
SUBNET_PREFIXES=( "${subnet_prefixes}" )
GATEWAYS=""
for subnet in $${SUBNET_PREFIXES[@]}
do
	gateway=$(python3 -c "import ipaddress;import sys;n = ipaddress.IPv4Network(sys.argv[1]);sys.stdout.write(n[1].compressed)" "$subnet")
	GATEWAYS="$GATEWAYS $gateway"
done
GATEWAYS=$(echo "$GATEWAYS" | sed 's/ //')

# get_core_ids bash function definition

core_ids=$(cat /sys/devices/system/cpu/cpu*/topology/thread_siblings_list | cut -d "-" -f 1 |  cut -d "," -f 1 | sort -u | tr '\n' ' ')
core_ids="$${core_ids[@]/0}"
IFS=', ' read -r -a core_ids <<< "$core_ids"
core_idx_begin=0
get_core_ids() {
	core_idx_end=$(($core_idx_begin + $1))
	res=$${core_ids["$core_idx_begin"]}
	for (( i=$(($core_idx_begin + 1)); i<$core_idx_end; i++ ))
	do
		res=$res,$${core_ids[i]}
	done
	core_idx_begin=$core_idx_end
	eval "$2=$res"
}

weka local stop
weka local rm default --force

# weka containers setup
get_core_ids $FRONTEND_CONTAINER_CORES_NUM frontend_core_ids
getNetStrForDpdk 1 $NICS_NUM "$GATEWAYS"

function retry_command {
  retry_max=60
  retry_sleep=30
  count=$retry_max
  command=$1
  msg=$2

  while [ $count -gt 0 ]; do
      $command && break
      count=$(($count - 1))
      echo "Retrying $msg in $retry_sleep seconds..."
      sleep $retry_sleep
  done
  [ $count -eq 0 ] && {
      echo "$msg failed after $retry_max attempts"
      echo "$(date -u):$msg failed"
      return 1
  }
  return 0
}

# changed standart frontend port to 14000 as it should be used locally for protocol setup:
# weka@ev-test-NFS-0:~$ weka nfs interface-group add test NFS
# error: Error: Failed connecting to http://127.0.0.1:14000/api/v1. Make sure weka is running on this host by running
# 	 weka local status | start
echo "$(date -u): setting up weka frontend"

run_container_cmd="weka local setup container --name frontend0 --base-port 14000 --cores $FRONTEND_CONTAINER_CORES_NUM --frontend-dedicated-cores $FRONTEND_CONTAINER_CORES_NUM --allow-protocols true --failure-domain $FAILURE_DOMAIN --core-ids $frontend_core_ids $net --dedicate --join-ips ${backend_lb_ip}"

retry_command "$run_container_cmd"  "setting up weka frontend"

echo "$(date -u): success to run weka frontend container"



# check that frontend container is up
ready_containers=0
while [ $ready_containers -ne 1 ];
do
  sleep 10
  ready_containers=$( weka local ps | grep frontend0 | grep -i 'running' | wc -l )
  echo "Running containers: $ready_containers"
done

echo "$(date -u): frontend is up"

# login to weka

# get token for key vault access
access_token=$(curl 'http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https%3A%2F%2Fvault.azure.net' -H Metadata:true | jq -r '.access_token')
# get key vault secret (get-weka-io-token)
weka_password=$(curl "${key_vault_url}secrets/weka-password?api-version=2016-10-01" -H "Authorization: Bearer $access_token" | jq -r '.value')



echo "$(date -u): try to run weka login command"

retry_command "weka user login admin $weka_password" "login to weka cluster"

echo "$(date -u): success to run weka login command"

rm -rf $INSTALLATION_PATH


echo "$(date -u): starting preparation for protocol setup"

weka local ps

current_mngmnt_ip=$(weka local resources | grep 'Management IPs' | awk '{print $NF}')

# get real primary ip from Azure cloud metadata
primary_ip=$(curl -s -H Metadata:true --noproxy "*" http://169.254.169.254/metadata/instance/network/interface/0/ipv4/ipAddress/0?api-version=2023-07-01 | jq -r '.privateIpAddress')

# get container id
max_retries=12 # 12 * 10 = 2 minutes
for ((i=0; i<max_retries; i++)); do
  container_id=$(weka cluster container | grep frontend0 | grep ${gateways_name} | grep $current_mngmnt_ip | grep UP | awk '{print $1}')
  if [ -n "$container_id" ]; then
      echo "$(date -u): frontend0 container id: $container_id"
      break
  fi
  echo "$(date -u): waiting for frontend0 container to be up"
  sleep 10
done
if [ -z "$container_id" ]; then
  echo "$(date -u): Failed to get the frontend0 container ID."
  exit 1
fi

# make primary ip the management ip for the weka container
if [ "$current_mngmnt_ip" != "$primary_ip" ]; then
  weka cluster container management-ips $container_id $primary_ip
  weka cluster container apply $container_id -f
  
  # wait for container to be up
  max_retries=12 # 12 * 10 = 2 minutes
  for ((i=0; i<max_retries; i++)); do
    status=$(weka cluster container $container_id | grep $container_id | awk '{print $5}')
    if [ "$status" == "UP" ]; then
    	echo "$(date -u): frontend0 container status: $status"
    	break
    fi
    echo "$(date -u): waiting for frontend0 container status to be UP, current status: $status"
    sleep 10
  done
  if [ "$status" != "UP" ]; then
    echo "$(date -u): failed to wait for the frontend0 container status to be UP"
    exit 1
  fi
fi

echo "$(date -u): finished preparation for protocol setup"
