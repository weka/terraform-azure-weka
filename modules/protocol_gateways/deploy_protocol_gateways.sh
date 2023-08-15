# get backend_ips using fetch function
max_retries=60 # 60 * 10 = 10 minutes
for (( i=0; i < max_retries; i++ )); do
  fetch_output=$(curl ${fetch_function_url}?code="${function_app_key}" --fail -H "Content-Type: application/json")
  backend_ips=($(echo "$fetch_output" | jq -r '.backend_ips[]'))
  # while backend_ips length is < cluster_size, keep fetching backend_ips
  if [ $${#backend_ips[@]} -lt ${cluster_size} ]; then
    echo "$(date -u): backend_ips length is $${#backend_ips[@]}, weka cluster_size is ${cluster_size}, fetching backend_ips again..."
    sleep 10
  fi
  echo "$(date -u): backend_ips: $${backend_ips[@]}"
  break
done
if (( i > max_retries )); then
    echo "$(date -u): timeout: unable to fetch all ${cluster_size} backend_ips after $max_retries attempts."
    exit 1
fi

FAILURE_DOMAIN=$(printf $(hostname -I) | sha256sum | tr -d '-' | cut -c1-16)
NUM_FRONTEND_CONTAINERS=${frontend_num}
NICS_NUM=${nics_num}
SUBNET_PREFIXES=( "${subnet_prefixes}" )
# get comma-separated list of backend ips
BACKEND_IPS=$(echo "$${backend_ips[@]}" | sed 's/ /,/g')
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
get_core_ids $NUM_FRONTEND_CONTAINERS frontend_core_ids

getNetStrForDpdk $(($NICS_NUM-1)) $(($NICS_NUM)) "$GATEWAYS" "$SUBNETS"

echo "$(date -u): setting up weka frontend"
# changed standart frontend port to 14000 as it should be used locally for protocol setup:
# weka@ev-test-NFS-0:~$ weka nfs interface-group add test NFS
# error: Error: Failed connecting to http://127.0.0.1:14000/api/v1. Make sure weka is running on this host by running
# 	 weka local status | start
sudo weka local setup container --name frontend0 --base-port 14000 --cores $NUM_FRONTEND_CONTAINERS --frontend-dedicated-cores $NUM_FRONTEND_CONTAINERS --allow-protocols true --failure-domain $FAILURE_DOMAIN --core-ids $frontend_core_ids $net --dedicate --join-ips $BACKEND_IPS


# check that frontend container is up
ready_containers=0
while [ $ready_containers -ne 1 ];
do
  sleep 10
  ready_containers=$( weka local ps | grep -i 'running' | wc -l )
  echo "Running containers: $ready_containers"
done

echo "$(date -u): frontend is up"

# login to weka

# get token for key vault access
access_token=$(curl 'http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https%3A%2F%2Fvault.azure.net' -H Metadata:true | jq -r '.access_token')
# get key vault secret (get-weka-io-token)
weka_password=$(curl "${key_vault_url}secrets/weka-password?api-version=2016-10-01" -H "Authorization: Bearer $access_token" | jq -r '.value')

weka user login admin $weka_password

rm -rf $INSTALLATION_PATH
