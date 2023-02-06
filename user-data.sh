#!/bin/bash
set -ex

# add private key to ssh agent
echo "${private_ssh_key}" > /home/${user}/.ssh/weka.pem
chmod 600 /home/${user}/.ssh/weka.pem
eval "$(ssh-agent -s)"
ssh-add /home/${user}/.ssh/weka.pem
rm -rf /home/${user}/.ssh/weka.pem

apt install net-tools -y

# set apt private repo
if [[ "${apt_repo_url}" ]]; then
  mv /etc/apt/sources.list /etc/apt/sources.list.bak
  echo "deb ${apt_repo_url} focal main restricted universe" > /etc/apt/sources.list
fi

INSTALLATION_PATH="/tmp/weka"
mkdir -p $INSTALLATION_PATH

# install ofed
OFED_NAME=ofed-${ofed_version}
if [[ "${install_ofed_url}" ]]; then
  wget ${install_ofed_url} -O $INSTALLATION_PATH/$OFED_NAME.tgz
else
  wget http://content.mellanox.com/ofed/MLNX_OFED-${ofed_version}/MLNX_OFED_LINUX-${ofed_version}-ubuntu18.04-x86_64.tgz -O $INSTALLATION_PATH/$OFED_NAME.tgz
fi

tar xf $INSTALLATION_PATH/$OFED_NAME.tgz --directory $INSTALLATION_PATH --one-top-level=$OFED_NAME
cd $INSTALLATION_PATH/$OFED_NAME/*/
./mlnxofedinstall --without-fw-update --add-kernel-support --force
/etc/init.d/openibd restart

apt update -y
apt install -y jq

# attache disk
disk=$(lsblk -o NAME,HCTL,SIZE,MOUNTPOINT | grep "3:0:0:0" | awk '{print $1}')
wekaiosw_device=/dev/$disk
mkfs.ext4 -L wekaiosw $wekaiosw_device || return 1
mkdir -p /opt/weka || return 1
mount $wekaiosw_device /opt/weka || return 1
echo "LABEL=wekaiosw /opt/weka ext4 defaults 0 2" >>/etc/fstab

rm -rf $INSTALLATION_PATH

curl ${deploy_url}?code="${function_app_default_key}" > /tmp/deploy.sh
chmod +x /tmp/deploy.sh
/tmp/deploy.sh > /tmp/weka_deploy.log 2>&1
