#!/bin/bash
set -ex

# set apt private repo
if [[ "${apt_repo_server}" ]]; then
  mv /etc/apt/sources.list /etc/apt/sources.list.bak
  echo "deb ${apt_repo_server} focal main restricted universe" > /etc/apt/sources.list
  echo "deb ${apt_repo_server} focal-updates main restricted" >> /etc/apt/sources.list
fi

for(( i=0; i<${nics_num}; i++ )); do
    cat <<-EOF | sed -i "/        eth$i/r /dev/stdin" /etc/netplan/50-cloud-init.yaml
            mtu: 3900
EOF
done

# config network with multi nics
echo "200 eth0-rt" >> /etc/iproute2/rt_tables

echo "network:"> /etc/cloud/cloud.cfg.d/99-disable-network-config.cfg
echo "  config: disabled" >> /etc/cloud/cloud.cfg.d/99-disable-network-config.cfg
gateway=$(ip r | grep default | awk '{print $3}')
eth=$(ifconfig | grep eth0 -C2 | grep 'inet ' | awk '{print $2}')
cat <<-EOF | sed -i "/            set-name: eth0/r /dev/stdin" /etc/netplan/50-cloud-init.yaml
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

netplan apply
