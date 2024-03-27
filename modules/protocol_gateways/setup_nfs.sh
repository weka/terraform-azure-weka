# get device to use
port=$(ip -o -f inet addr show | grep "$current_mngmnt_ip/"| awk '{print $2}')
subnet_mask=$(ifconfig $port | grep 'inet ' | awk '{print $4}')
gateway=$(ip r | grep default | awk '{print $3}')
# get array of secondary ips (excluding the primary ip)
secondary_ips=($(ip -o -4 addr show dev $port | awk '{print $4}' | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | grep -v $primary_ip))

echo "$(date -u): current_mngmnt_ip: $current_mngmnt_ip"
echo "$(date -u): primary_ip: $primary_ip"
echo "$(date -u): secondary_ips: $${secondary_ips[@]}"

function create_interface_group() {
  if weka nfs interface-group | grep ${interface_group_name}; then
    echo "$(date -u): interface group ${interface_group_name} already exists"
    return
  fi
  echo "$(date -u): creating interface group"
  weka nfs interface-group add ${interface_group_name} NFS --subnet $subnet_mask --gateway $gateway
  echo "$(date -u): interface group ${interface_group_name} created"
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
      echo "$(date -u): timeout: weka filesystem $filesystem_name is not up after $max_retries attempts."
      return 1
  fi
}

function create_client_group() {
  if weka nfs client-group | grep ${client_group_name}; then
    echo "$(date -u): client group ${client_group_name} already exists"
    return
  fi
  echo "$(date -u): creating client group"
  weka nfs client-group add ${client_group_name}
  weka nfs rules add dns ${client_group_name} *
  wait_for_weka_fs || return 1
  weka nfs permission add default ${client_group_name}
  echo "$(date -u): client group ${client_group_name} created"
}

# make sure weka cluster is already up
max_retries=60 # 60 * 30 = 30 minutes
for (( i=0; i < max_retries; i++ )); do
  if [ $(weka status | grep 'status: OK' | wc -l) -ge 1 ]; then
    echo "$(date -u): weka cluster is up"
    break
  fi
  echo "$(date -u): waiting for weka cluster to be up"
  sleep 30
done
if (( i > max_retries )); then
    echo "$(date -u): timeout: weka cluster is not up after $max_retries attempts."
    exit 1
fi

# create interface group if not exists
create_interface_group || true

# add “port” to the interface group - basically it means adding a host and its net device to the group
weka nfs interface-group port add ${interface_group_name} $container_id $port
# show interface group
weka nfs interface-group


function wait_for_nfs_interface_group(){
  max_retries=12 # 12 * 10 = 2 minutes
  for ((i=0; i<max_retries; i++)); do
    status=$(weka nfs interface-group -J | jq -r '.[] | select(.name == "'${interface_group_name}'").status')
    if [ "$status" == "OK" ]; then
        echo "$(date -u): interface group status: $status"
        break
    fi
    echo "$(date -u): waiting for interface group status to be OK, current status: $status"
    sleep 10
  done
  if [ "$status" != "OK" ]; then
    echo "$(date -u): failed to wait for the interface group status to be OK"
    return 1
  fi
}

# add secondary IPs for the group to use - these IPs will be used in order to mount
for seconday_ip in "$${secondary_ips[@]}"; do
  wait_for_nfs_interface_group || exit 1
  # add secondary ip to the interface group
  retry_command "weka nfs interface-group ip-range add ${interface_group_name} $seconday_ip"

  wait_for_nfs_interface_group || exit 1
done

weka nfs interface-group

# create client group if not exists and add rules / premissions
create_client_group || true

weka nfs client-group

echo "$(date -u): NFS setup complete"
