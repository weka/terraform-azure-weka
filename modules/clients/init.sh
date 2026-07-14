#!/bin/bash
set -ex

# set apt private repo
if [[ "${apt_repo_server}" ]]; then
  mv /etc/apt/sources.list /etc/apt/sources.list.bak
  echo "deb ${apt_repo_server} focal main restricted universe" > /etc/apt/sources.list
  echo "deb ${apt_repo_server} focal-updates main restricted" >> /etc/apt/sources.list
fi

if ! command -v jq &> /dev/null; then
  apt install -y jq
fi

# config network with multi nics
echo "200 eth0-rt" >> /etc/iproute2/rt_tables

gateway=$(ip r | grep default | awk '{print $3}')
eth=$(ip -4 addr show eth0 | awk '/inet / {split($2,a,"/"); print a[1]}')

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

# set MTU on the remaining NICs
for(( i=1; i<${nics_num}; i++ )); do
    cat >> /etc/netplan/99-weka.yaml <<EOF
    eth$i:
      mtu: 3900
EOF
done

chmod 600 /etc/netplan/99-weka.yaml
netplan apply
