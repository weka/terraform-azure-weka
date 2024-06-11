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

are_routes_ready='ip route | grep eth1'
for(( i=2; i<${nics_num}; i++ )); do
  are_routes_ready=$are_routes_ready' && ip route | grep eth'"$i"
done
cat >>/usr/sbin/remove-routes.sh <<EOF
#!/bin/bash
set -ex
retry_max=24
for(( i=0; i<\$retry_max; i++ )); do
  if eval "$are_routes_ready"; then
    for(( j=1; j<${nics_num}; j++ )); do
      /usr/sbin/ip route del ${subnet_range} dev eth\$j
    done
    break
  fi
  ip route
  sleep 5
done
if [ \$i -eq \$retry_max ]; then
  echo "Routes are not ready on time"
  shutdown -h now
  exit 1
fi
echo "Routes were removed successfully"
EOF

chmod +x /usr/sbin/remove-routes.sh

cat >/etc/systemd/system/remove-routes.service <<EOF
[Unit]
Description=Remove specific routes
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/bin/bash /usr/sbin/remove-routes.sh

[Install]
WantedBy=multi-user.target
EOF

ip route # show routes before removing
systemctl daemon-reload
systemctl enable remove-routes.service
systemctl start remove-routes.service
systemctl status remove-routes.service || true # show status of remove-routes.service
ip route # show routes after removing

echo "$(date -u): routes configured"

while ! [ "$(lsblk | grep ${disk_size}G | awk '{print $1}')" ] ; do
  echo "waiting for disk to be ready"
  sleep 5
done

compute_name=$(curl -s -H Metadata:true --noproxy "*" "http://169.254.169.254/metadata/instance?api-version=2021-02-01" | jq '.compute.name')
compute_name=$(echo "$compute_name" | cut -c2- | rev | cut -c2- | rev)
retry=0
while ! curl ${deploy_url}?code="${function_app_default_key}" --fail -H "Content-Type:application/json" -d "{\"name\": \"$compute_name:$HOSTNAME\", \"protocol\": \"${protocol}\"}" > /tmp/deploy.sh 2>/tmp/deploy_err.log || [ ! -s /tmp/deploy.sh ]; do
  echo "Retry $retry: waiting for deploy script generation success"
  cat /tmp/deploy_err.log
  retry=$((retry + 1))
  sleep 5
done

weka_dir="/opt/weka/data"
mkdir -p $weka_dir
mv /root/weka-prepackaged $weka_dir

if [ $retry -gt 0 ]; then
  msg="Deploy script generation retried $retry times"
  echo "$msg"
  curl -i "${report_url}?code=${function_app_default_key}" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"protocol\": \"${protocol}\", \"type\": \"debug\", \"message\": \"$msg\"}"
fi

echo "$(date -u): running deploy script"

chmod +x /tmp/deploy.sh
/tmp/deploy.sh 2>&1 | tee /tmp/weka_deploy.log
