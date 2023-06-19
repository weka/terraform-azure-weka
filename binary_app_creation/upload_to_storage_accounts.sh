#!/bin/bash

# Usage:
# upload_to_storage_accounts.sh <regions_file_dir> <dist> <os_name> <code_path> <binary_dir> <resource_group>

regions_file_dir="$1"
dist="$2"
os_name="$3"
code_path="$4"
binary_dir="$5"
resource_group="$6"

regions_file="$regions_file_dir/${dist}.txt"
current_script_dir=$(dirname ${BASH_SOURCE[0]})
code_hash="$($current_script_dir/get_code_hash.sh ${os_name} ${code_path})"

bin_path="$binary_dir/${code_hash}"
az_filename="${DIST}/${code_hash}"

while read region; do 
    echo "Uploading to region: $region"

    storage_account="weka${region}"
    container_name="weka-tf-bin-deployment-${region}"

    ./binary_app_creation/upload_to_azure_storage.sh $bin_path $resource_group $storage_account $container_name $az_filename
done < $regions_file
