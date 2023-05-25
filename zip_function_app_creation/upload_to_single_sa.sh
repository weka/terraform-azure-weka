#!/bin/bash

# Usage:
# upload_to_single_sa.sh <os_name> <function_code_path> <function_binaries_dir> <resource_group> <storage_account> <continer_name>

os_name="$1"
function_code_path="$2"
function_binaries_dir="$3"
resource_group="$4"
storage_account="$5"
container_name="$6"

current_script_dir=$(dirname ${BASH_SOURCE[0]})
function_app_code_hash="$($current_script_dir/get_function_app_hash.sh ${os_name} ${function_code_path})"

file_path="$function_binaries_dir/${function_app_code_hash}/weka-deployment"
az_filename="${function_app_code_hash}/weka-deployment"

echo "Uploading to Storage Account: $storage_account"
${current_script_dir}/upload_to_azure_storage.sh $file_path $resource_group $storage_account $container_name $az_filename
