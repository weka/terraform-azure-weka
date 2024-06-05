echo "$(date -u): running smb script"

weka local ps

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

# new smbw config, where smbw is the default
smb_cmd_extention=""
if [[ ${smbw_enabled} == false ]]; then
    smb_cmd_extention="--smb"
fi

function handle_cluster_create_output() {
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

function create_old_smb_cluster() {
  echo "$(date -u): trying to create old SMB cluster"
  cluster_create_output=$(weka smb cluster create ${cluster_name} ${domain_name} $smbw_cmd_extention --container-ids $all_container_ids_str 2>&1)
  handle_cluster_create_output cluster_create_output
}

function create_new_smb_cluster() {
  echo "$(date -u): trying to create new SMB cluster"
  cluster_create_output=$(weka smb cluster create ${cluster_name} ${domain_name} .config_fs --container-ids $all_container_ids_str $smb_cmd_extention 2>&1)
  handle_cluster_create_output cluster_create_output
}

create_old_smb_cluster || create_new_smb_cluster || exit 1

weka smb cluster wait

weka smb cluster status

echo "$(date -u): SMB cluster is created successfully"
