#!/bin/bash
set -ex

curl -i ${report_url}?code="${function_app_default_key}" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"Running init script\"}"

handle_error () {
  if [ "$1" -ne 0 ]; then
    curl -i ${report_url}?code="${function_app_default_key}" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"error\", \"message\": \"${2}\"}"
    exit 1
  fi
}

# add private key to ssh config
echo "${private_ssh_key}" > /home/${user}/.ssh/weka.pem
chmod 600 /home/${user}/.ssh/weka.pem

cat > /home/${user}/.ssh/config <<EOL
Host *
   User ${user}
   IdentityFile /home/${user}/.ssh/weka.pem
EOL

cp -R /home/${user}/.ssh/* /root/.ssh/
chown -R ${user}:${user} /home/${user}/.ssh/

while fuser /var/{lib/{dpkg,apt/lists},cache/apt/archives}/lock >/dev/null 2>&1; do
   sleep 2
done
apt install net-tools -y

# set apt private repo
if [[ "${apt_repo_url}" ]]; then
  mv /etc/apt/sources.list /etc/apt/sources.list.bak
  echo "deb ${apt_repo_url} focal main restricted universe" > /etc/apt/sources.list
fi

INSTALLATION_PATH="/tmp/weka"
mkdir -p $INSTALLATION_PATH

# install ofed
if [[ "${skip_ofed_installation}" == false ]]; then
  curl -i ${report_url}?code="${function_app_default_key}" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"installing ofed\"}"
  OFED_NAME=ofed-${ofed_version}
  if [[ "${install_ofed_url}" ]]; then
    wget "${install_ofed_url}" -O $INSTALLATION_PATH/$OFED_NAME.tgz
  else
    wget http://content.mellanox.com/ofed/MLNX_OFED-${ofed_version}/MLNX_OFED_LINUX-${ofed_version}-ubuntu18.04-x86_64.tgz -O $INSTALLATION_PATH/$OFED_NAME.tgz
  fi

  tar xf $INSTALLATION_PATH/$OFED_NAME.tgz --directory $INSTALLATION_PATH --one-top-level=$OFED_NAME
  cd $INSTALLATION_PATH/$OFED_NAME/*/
  ./mlnxofedinstall --without-fw-update --add-kernel-support --force 2>&1 | tee /tmp/weka_ofed_installation
  /etc/init.d/openibd restart

  curl -i ${report_url}?code="${function_app_default_key}" -H "Content-Type:application/json" -d "{\"hostname\": \"$HOSTNAME\", \"type\": \"progress\", \"message\": \"ofed installation completed\"}"
fi

apt update -y
apt install -y jq

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
