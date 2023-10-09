#!/bin/bash

ofed_version=$1
subscription_id=$2

account_name="wekadeploytars"
container_name="tars"
path_location="/tmp"
resource_group="weka-tf"

if [ $# -eq 0 ]; then
    echo "No arguments provided!!! Please run as ./upload_ofed_to_blob.sh OFED_VERSION SUBSCRIPTION_ID"
    exit 1
fi

echo "=====> Download ofed tgz..."
if test -f "${path_location}/MLNX_OFED_LINUX-${ofed_version}-ubuntu18.04-x86_64.tgz"; then
    echo "ofed tar file already exists in folder ${path_location}/MLNX_OFED_LINUX-${ofed_version}-ubuntu18.04-x86_64.tgz..."
else
  wget wget http://content.mellanox.com/ofed/MLNX_OFED-${ofed_version}/MLNX_OFED_LINUX-${ofed_version}-ubuntu18.04-x86_64.tgz  -P ${path_location}
fi

echo "=====> Get Storage account key..."
account_key=`az storage account keys list -g ${resource_group} -n ${account_name} --subscription ${subscription_id} --query [0].value -o tsv`

echo "=====> Upload tgz to blob..."
az storage blob upload \
    --account-name ${account_name} \
    --container-name ${container_name} \
    --name MLNX_OFED_LINUX-${ofed_version}-ubuntu18.04-x86_64.tgz \
    --file /tmp/MLNX_OFED_LINUX-${ofed_version}-ubuntu18.04-x86_64.tgz \
    --account-key ${account_key} --subscription ${subscription_id}
