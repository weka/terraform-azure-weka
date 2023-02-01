#!/bin/bash

weka_token=$1
weka_version=$2
subscription_id=$3

account_name="wekadeploytars"
container_name="tars"
path_location="/tmp"
resource_group="weka-tf"

if [ $# -eq 0 ]; then
    echo "No arguments provided!!! Please run as ./upload_tar_to_blob.sh WEKA_TOKEN WEKA_VERSION SUBSCRIPTION_ID"
    exit 1
fi

weka_tar="weka-${weka_version}.tar"
echo "=====> Download weka tar..."
if test -f "${path_location}/${weka_version}.tar"; then
    echo "weka tar file already exists in folder ${path_location}/${weka_tar}..."
else
  wget --auth-no-challenge "https://${weka_token}:@get.prod.weka.io/dist/v1/pkg/weka-${weka_version}.tar" -P ${path_location}
fi

echo "=====> Get Storage account key..."
account_key=`az storage account keys list -g ${resource_group} -n ${account_name} --subscription ${subscription_id}  --query [0].value -o tsv`

echo "=====> Upload tar to blob..."
az storage blob upload \
    --account-name ${account_name} \
    --container-name ${container_name} \
    --name ${weka_tar} \
    --file /tmp/${weka_tar} \
    --account-key ${account_key} --subscription ${subscription_id}