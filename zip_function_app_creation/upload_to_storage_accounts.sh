#!/bin/bash

# Usage:
# upload_to_storage_accounts.sh <regions_file_dir> <dist> <os_name> <function_code_path> <function_zip_dir> <resource_group>

regions_file_dir="$1"
dist="$2"
os_name="$3"
function_code_path="$4"
function_zip_dir="$5"
resource_group="$6"

regions_file="$regions_file_dir/${dist}.txt"
current_script_dir=$(dirname ${BASH_SOURCE[0]})
function_app_code_hash="$($current_script_dir/get_function_app_hash.sh ${os_name} ${function_code_path})"

zip_path="$function_zip_dir/${function_app_code_hash}.zip"
az_filename="${DIST}/${function_app_code_hash}.zip"

while read region; do
    echo "Uploading to region: $region"

    storage_account="weka${region}"
    container_name="weka-tf-functions-deployment-${region}"

    ./zip_function_app_creation/upload_to_azure_storage.sh $zip_path $resource_group $storage_account $container_name $az_filename
done < $regions_file
