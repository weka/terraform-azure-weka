echo "$(date -u): running smb script"
weka local ps

# get token for key vault access
access_token=$(curl 'http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https%3A%2F%2Fvault.azure.net' -H Metadata:true | jq -r '.access_token')
# get key vault secret
function_app_key=$(curl "${key_vault_url}secrets/${vault_function_app_key_name}?api-version=2016-10-01" -H "Authorization: Bearer $access_token" | jq -r '.value')

function report {
  local json_data=$1
  curl ${report_function_url}?code="$function_app_key" -H 'Content-Type:application/json' -d "$json_data"
}

function wait_for_weka_fs(){
  filesystem_name="default"
  max_retries=30 # 30 * 10 = 5 minutes
  for (( i=0; i < max_retries; i++ )); do
    if [ "$(weka fs | grep -c $filesystem_name)" -ge 1 ]; then
      echo "$(date -u): weka filesystem $filesystem_name is up"
      break
    fi
    echo "$(date -u): waiting for weka filesystem $filesystem_name to be up"
    sleep 10
  done
  if (( i > max_retries )); then
      err_msg="timeout: weka filesystem $filesystem_name is not up after $max_retries attempts."
      echo "$(date -u): $err_msg"
      report "{\"hostname\": \"$HOSTNAME\", \"type\": \"error\", \"message\": \"$err_msg\"}"
      return 1
  fi
}

function create_config_fs(){
  filesystem_name=".config_fs"
  size="10GB"

  if [ "$(weka fs | grep -c $filesystem_name)" -ge 1 ]; then
    echo "$(date -u): weka filesystem $filesystem_name exists"
    return 0
  fi

  echo "$(date -u): trying to create filesystem $filesystem_name"
  output=$(weka fs create $filesystem_name default $size 2>&1)
  # possiible outputs:
  # FSId: 1 (means success)
  # error: The given filesystem ".config_fs" already exists.
  # error: Not enough available drive capacity for filesystem. requested "10.00 GB", but only "0 B" are free
  if [ $? -eq 0 ]; then
    echo "$(date -u): weka filesystem $filesystem_name is created"
    return 0
  fi

  if [[ $output == *"already exists"* ]]; then
    echo "$(date -u): weka filesystem $filesystem_name already exists"
    break
  elif [[ $output == *"Not enough available drive capacity for filesystem"* ]]; then
    err_msg="Not enough available drive capacity for filesystem $filesystem_name for size $size"
    echo "$(date -u): $err_msg"
    report "{\"hostname\": \"$HOSTNAME\", \"type\": \"error\", \"message\": \"$err_msg\"}"
    return 1
  else
    echo "$(date -u): output: $output"
    report "{\"hostname\": \"$HOSTNAME\", \"type\": \"error\", \"message\": \"cannot create weka filesystem $filesystem_name\"}"
    return 1
  fi
}

if [[ ${smbw_enabled} == true ]]; then
  wait_for_weka_fs || exit 1
  create_config_fs || exit 1
fi

# make sure weka cluster is already up
max_retries=60
for (( i=0; i < max_retries; i++ )); do
  if [ $(weka status | grep 'status: OK' | wc -l) -ge 1 ]; then
    echo "$(date -u): weka cluster is up"
    break
  fi
  echo "$(date -u): waiting for weka cluster to be up"
  sleep 30
done
if (( i > max_retries )); then
    err_msg="timeout: weka cluster is not up after $max_retries attempts."
    echo "$(date -u): $err_msg"
    report "{\"hostname\": \"$HOSTNAME\", \"type\": \"error\", \"message\": \"$err_msg\"}"
    exit 1
fi

cluster_size="${gateways_number}"

current_mngmnt_ip=$(weka local resources | grep 'Management IPs' | awk '{print $NF}')
# get container id
for ((i=0; i<20; i++)); do
  container_id=$(weka cluster container | grep frontend0 | grep ${gateways_name} | grep $current_mngmnt_ip | grep UP | awk '{print $1}')
  if [ -n "$container_id" ]; then
      echo "$(date -u): frontend0 container id: $container_id"
      report "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"frontend0 container $container_id is up\"}"
      break
  fi
  echo "$(date -u): waiting for frontend0 container to be up"
  sleep 5
done

if [ -z "$container_id" ]; then
  err_msg="Failed to get the frontend0 container ID."
  echo "$(date -u): $err_msg"
  report "{\"hostname\": \"$HOSTNAME\", \"type\": \"error\", \"message\": \"$err_msg\"}"
  exit 1
fi

# wait for all containers to be ready
max_retries=60
for (( retry=1; retry<=max_retries; retry++ )); do
    # get all UP gateway container ids
    all_container_ids=$(weka cluster container | grep frontend0 | grep ${gateways_name} | grep UP | awk '{print $1}')
    # if number of all_container_ids < cluster_size, do nothing
    all_container_ids_number=$(echo "$all_container_ids" | wc -l)
    if (( all_container_ids_number < cluster_size )); then
        echo "$(date -u): not all containers are ready - do retry $retry of $max_retries"
        sleep 20
    else
        echo "$(date -u): all containers are ready"
        break
    fi
done

if (( retry > max_retries )); then
    err_msg="timeout: not all containers are ready after $max_retries attempts."
    echo "$(date -u): $err_msg"
    report "{\"hostname\": \"$HOSTNAME\", \"type\": \"error\", \"message\": \"$err_msg\"}"
    exit 1
fi

# wait for weka smb cluster to be ready in case it was created by another host
weka smb cluster wait

not_ready_hosts=$(weka smb cluster status | grep 'Not Ready' | wc -l)
all_hosts=$(weka smb cluster status | grep 'Host' | wc -l)

if (( all_hosts > 0 && not_ready_hosts == 0 && all_hosts == cluster_size )); then
    echo "$(date -u): SMB cluster is already created"
    weka smb cluster status
    exit 0
fi

if (( all_hosts > 0 && not_ready_hosts == 0 && all_hosts < cluster_size )); then
    echo "$(date -u): SMB cluster already exists, adding current container to it"

    weka smb cluster containers add --container-ids $container_id
    weka smb cluster wait
    weka smb cluster status
    exit 0
fi

echo "$(date -u): weka SMB cluster does not exist, creating it"
# get all protocol gateways fromtend container ids separated by comma
all_container_ids_str=$(echo "$all_container_ids" | tr '\n' ',' | sed 's/,$//')

sleep 30s
# if smbw_enabled is true, enable SMBW by adding --smbw flag
smbw_cmd_extention=""
if [[ ${smbw_enabled} == true ]]; then
    smbw_cmd_extention="--smbw --config-fs-name .config_fs"
fi

function create_smb_cluster {
  cluster_create_output=$(weka smb cluster create ${cluster_name} ${domain_name} $smbw_cmd_extention --container-ids $all_container_ids_str 2>&1)

  if [ $? -eq 0 ]; then
    msg="SMB cluster is created"
    echo "$(date -u): $msg"
    report "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"$msg\"}"
    return 0
  elif [[ $cluster_create_output == *"Cluster is already configured"* ]]; then
    msg="SMB cluster is already configured"
    echo "$(date -u): $msg"
    report "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"$msg\"}"
    weka smb cluster status
    return 0
  else
    echo "$(date -u): $cluster_create_output"
    report "{\"hostname\": \"$HOSTNAME\", \"type\": \"error\", \"message\": \"$cluster_create_output\"}"
    return 1
  fi
}

create_smb_cluster || exit 1

weka smb cluster wait

# add an SMB share if share_name is not empty
# 'default' is the fs-name of weka file system created during clusterization
if [ -n "${share_name}" ]; then
    weka smb share add ${share_name} default || true
fi

weka smb cluster status

echo "$(date -u): SMB cluster is created successfully"
