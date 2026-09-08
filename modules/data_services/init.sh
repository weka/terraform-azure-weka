#!/bin/bash
set -ex

echo "$(date -u): cloud-init beginning"

# set apt private repo
if [[ "${apt_repo_server}" != "" ]]; then
  mv /etc/apt/sources.list /etc/apt/sources.list.bak
  echo "deb ${apt_repo_server} focal main restricted universe" > /etc/apt/sources.list
  echo "deb ${apt_repo_server} focal-updates main restricted" >> /etc/apt/sources.list
  apt update -y
fi

if ! command -v jq &> /dev/null; then
  apt install -y jq
fi

# data services machines have a single NIC, so only eth0 needs to be configured

gateway=$(ip r | grep default | awk '{print $3}')
eth=$(ip -4 addr show eth0 | awk '/inet / {split($2,a,"/"); print a[1]}')

echo "200 eth0-rt" >> /etc/iproute2/rt_tables

# Write weka network config to a dedicated netplan file instead of editing
# 50-cloud-init.yaml. netplan is declarative, so it is re-applied on every boot
# and survives reboots on its own. The 99- prefix also means it wins over
# cloud-init's 50- file if that ever gets regenerated.
cat > /etc/netplan/99-weka.yaml <<EOF
network:
  version: 2
  ethernets:
    eth0:
      mtu: 3900
      routes:
       - to: ${subnet_range}
         via: $gateway
         metric: 200
         table: 200
       - to: 0.0.0.0/0
         via: $gateway
         table: 200
      routing-policy:
       - from: $eth/32
         table: 200
       - to: $eth/32
         table: 200
EOF

chmod 600 /etc/netplan/99-weka.yaml
netplan apply

echo "$(date -u): network configured"

while ! [ "$(lsblk | grep ${disk_size}G | awk '{print $1}')" ] ; do
  echo "waiting for disk to be ready"
  sleep 5
done

compute_name=$(curl -s -H Metadata:true --noproxy "*" "http://169.254.169.254/metadata/instance?api-version=2021-02-01" | jq -r '.compute.name')
retry=0
while ! curl ${deploy_url}?code="${function_app_default_key}" --fail -H "Content-Type:application/json" -d "{\"name\": \"$compute_name:$HOSTNAME\", \"protocol\": \"data\"}" > /tmp/deploy.sh 2>/tmp/deploy_err.log || [ ! -s /tmp/deploy.sh ]; do
  echo "Retry $retry: waiting for deploy script generation success"
  cat /tmp/deploy_err.log
  retry=$((retry + 1))
  sleep 5
done

if [ -d "/root/weka-prepackaged" ]; then
  weka_dir="/opt/weka/data"
  mkdir -p $weka_dir
  mv /root/weka-prepackaged $weka_dir
fi

if [ $retry -gt 0 ]; then
  msg="Deploy script generation retried $retry times"
  echo "$msg"
  curl -i "${report_url}?code=${function_app_default_key}" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"protocol\": \"data\", \"type\": \"debug\", \"message\": \"$msg\"}"
fi

echo "$(date -u): running deploy script"

chmod +x /tmp/deploy.sh
/tmp/deploy.sh 2>&1 | tee /tmp/weka_deploy.log
