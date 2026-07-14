package scale_up

import "fmt"

var (
	initScript = `#!/bin/bash
set -ex

# user data
%s

DISK_SIZE=%d
NICS_NUM=%d
SUBNET_RANGE="%s"
APT_REPO_SERVER="%s"

# report function definition
%s

# deploy function definition
%s

report "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Running init script\"}"

while fuser /var/{lib/{dpkg,apt/lists},cache/apt/archives}/lock >/dev/null 2>&1; do
   sleep 2
done

# set apt private repo
if [[ "$APT_REPO_SERVER" ]]; then
  mv /etc/apt/sources.list /etc/apt/sources.list.bak
  echo "deb $APT_REPO_SERVER focal main restricted universe" > /etc/apt/sources.list
  echo "deb $APT_REPO_SERVER focal-updates main restricted" >> /etc/apt/sources.list
  apt update -y
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
# and survives reboots on its own - no systemd service needed. The 99- prefix
# also means it wins over cloud-init's 50- file if that ever gets regenerated.
cat > /etc/netplan/99-weka.yaml <<EOF
network:
  version: 2
  ethernets:
    eth0:
      mtu: 3900
      routes:
       - to: $SUBNET_RANGE
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
for(( i=1; i<$NICS_NUM; i++ )); do
    cat >> /etc/netplan/99-weka.yaml <<EOF
    eth$i:
      mtu: 3900
EOF
done

chmod 600 /etc/netplan/99-weka.yaml
netplan apply

disk_size_str="${DISK_SIZE}G"
while ! [ "$(lsblk | grep $disk_size_str | awk '{print $1}')" ] ; do
  echo "waiting for disk to be ready"
  sleep 5
done

compute_name=""
max_retries=10
retry=0
while [ -z "$compute_name" ] && [ $retry -lt $max_retries ]; do
  compute_name=$(curl -s -H "Metadata:true" --noproxy "*" "http://169.254.169.254/metadata/instance?api-version=2021-02-01" | jq -r '.compute.name')

  if [ -z "$compute_name" ]; then
    echo "Attempt $((retry + 1)) to get compute name from metadata failed. Retrying in 3 seconds..."
    retry=$((retry + 1))
    sleep 3
  fi
done

if [ -z "$compute_name" ]; then
  echo "Failed to get compute name from metadata after $max_retries attempts"
  report "{\"hostname\": \"$HOSTNAME\", \"type\": \"error\", \"message\": \"Failed to get compute name from metadata\"}"
  shutdown -h now
  exit 1
fi

retry=0
while ! deploy "{\"name\": \"$compute_name:$HOSTNAME\"}" > /tmp/deploy.sh 2>/tmp/deploy_err.log || [ ! -s /tmp/deploy.sh ]; do
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
  report "{\"hostname\": \"$HOSTNAME\", \"type\": \"debug\", \"message\": \"$msg\"}"
fi

# debug info
ip route
ip addr show

echo "$(date -u): running deploy script"

chmod +x /tmp/deploy.sh
/tmp/deploy.sh 2>&1 | tee /tmp/weka_deploy.log

# debug info
ip route
ip addr show

# Install WEKA maintenance event monitor
echo "$(date -u): installing weka maintenance event monitor"

CLUSTER_NAME="%s"

# Create maintenance monitor script with injected fetch function
cat > /usr/local/bin/weka-maintenance-monitor.sh << 'MONITOR_SCRIPT_EOF'
%s
MONITOR_SCRIPT_EOF

chmod +x /usr/local/bin/weka-maintenance-monitor.sh

# Create environment configuration
cat > /etc/default/weka-maintenance-monitor << EOF
CLUSTER_NAME=$CLUSTER_NAME
CHECK_INTERVAL=30
EOF

chmod 644 /etc/default/weka-maintenance-monitor

# Create systemd service
cat > /etc/systemd/system/weka-maintenance-monitor.service << 'SERVICE_EOF'
%s
SERVICE_EOF

chmod 644 /etc/systemd/system/weka-maintenance-monitor.service

# Enable and start service
systemctl daemon-reload
systemctl enable weka-maintenance-monitor.service
systemctl start weka-maintenance-monitor.service

echo "$(date -u): weka maintenance event monitor installed and started"
report "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Weka maintenance event monitor installed and started\"}"
`
)

func getInitScript(userData string, diskSize int, nicsNum int, subnetRange string, aptRepoServer string, reportFuncDef string, deployFuncDef string, clusterName string, monitorScript string, serviceUnit string) string {
	return fmt.Sprintf(initScript, userData, diskSize, nicsNum, subnetRange, aptRepoServer, reportFuncDef, deployFuncDef, clusterName, monitorScript, serviceUnit)
}
