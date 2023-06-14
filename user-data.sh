#!/bin/bash
set -ex

curl -i ${report_url}?code="${function_app_default_key}" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Running init script\"}"

handle_error () {
  if [ "$1" -ne 0 ]; then
    curl -i ${report_url}?code="${function_app_default_key}" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"error\", \"message\": \"${2}\"}"
    exit 1
  fi
}

while fuser /var/{lib/{dpkg,apt/lists},cache/apt/archives}/lock >/dev/null 2>&1; do
   sleep 2
done

apt update -y

# set apt private repo
if [[ "${apt_repo_url}" ]]; then
  mv /etc/apt/sources.list /etc/apt/sources.list.bak
  echo "deb ${apt_repo_url} focal main restricted universe" > /etc/apt/sources.list
  echo "deb ${apt_repo_url} focal-updates main restricted" >> /etc/apt/sources.list
fi

INSTALLATION_PATH="/tmp/weka"
mkdir -p $INSTALLATION_PATH

for(( i=0; i<${nics_num}; i++ )); do
    cat <<-EOF | sed -i "/        eth$i/r /dev/stdin" /etc/netplan/50-cloud-init.yaml
            mtu: 3900
EOF
done

# config network with multi nics
if [[ ${install_cluster_dpdk} == true ]]; then
  for(( i=0; i<${nics_num}; i++)); do
    echo "20$i eth$i-rt" >> /etc/iproute2/rt_tables
  done

  echo "network:"> /etc/cloud/cloud.cfg.d/99-disable-network-config.cfg
  echo "  config: disabled" >> /etc/cloud/cloud.cfg.d/99-disable-network-config.cfg
  gateway=$(ip r | grep default | awk '{print $3}')
  for(( i=0; i<${nics_num}; i++ )); do
    eth=$(ifconfig | grep eth$i -C2 | grep 'inet ' | awk '{print $2}')
    cat <<-EOF | sed -i "/            set-name: eth$i/r /dev/stdin" /etc/netplan/50-cloud-init.yaml
            routes:
             - to: ${subnet_range}
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
fi

netplan apply

# attache disk
wekaiosw_device=/dev/"$(lsblk | grep ${disk_size}G | awk '{print $1}')"

status=0
mkfs.ext4 -L wekaiosw $wekaiosw_device 2>&1 | tee /tmp/output  || status=$?
handle_error $status "$(cat /tmp/output)"
mkdir -p /opt/weka 2>&1 | tee /tmp/output || status=$?
handle_error $status "$(cat /tmp/output)"
mount $wekaiosw_device /opt/weka  2>&1 | tee /tmp/output || status=$?
handle_error $status "$(cat /tmp/output)"
rm /tmp/output

echo "LABEL=wekaiosw /opt/weka ext4 defaults 0 2" >>/etc/fstab

rm -rf $INSTALLATION_PATH

compute_name=$(curl -s -H Metadata:true --noproxy "*" "http://169.254.169.254/metadata/instance?api-version=2021-02-01" | jq '.compute.name')
compute_name=$(echo "$compute_name" | cut -c2- | rev | cut -c2- | rev)
curl ${deploy_url}?code="${function_app_default_key}" --fail -H "Content-Type:application/json" -d "{\"vm\": \"$compute_name:$HOSTNAME\"}" > /tmp/deploy.sh
chmod +x /tmp/deploy.sh
/tmp/deploy.sh 2>&1 | tee /tmp/weka_deploy.log
