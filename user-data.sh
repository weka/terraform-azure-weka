#!/bin/bash
set -ex

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
wekaiosw_device=/dev/sdc
mkfs.ext4 -L wekaiosw $wekaiosw_device || return 1
mkdir -p /opt/weka || return 1
mount $wekaiosw_device /opt/weka || return 1
echo "LABEL=wekaiosw /opt/weka ext4 defaults 0 2" >>/etc/fstab

# install weka
WEKA_NAME=${weka_version}
if [[ "${install_weka_url}" ]]; then
    wget ${install_weka_url} -O $INSTALLATION_PATH/$WEKA_NAME.tar
    tar -xvf $INSTALLATION_PATH/$WEKA_NAME.tar --directory $INSTALLATION_PATH --one-top-level=$WEKA_NAME
    cd $INSTALLATION_PATH/$WEKA_NAME/weka-$WEKA_NAME
    ./install.sh
else
  curl https://${weka_token}@get.prod.weka.io/dist/v1/install/${weka_version}/${weka_version} | sh
fi

rm -rf $INSTALLATION_PATH

weka local stop
weka local rm default --force
weka local setup container --name drives0 --base-port 14000 --cores ${num_drive_containers} --no-frontends --drives-dedicated-cores ${num_drive_containers}

curl ${clusterization_url}?code="${function_app_default_key}" -H "Content-Type:application/json"  -d "{\"name\": \"$HOSTNAME\"}" > /tmp/clusterize.sh
chmod +x /tmp/clusterize.sh
/tmp/clusterize.sh > /tmp/cluster_creation.log 2>&1
