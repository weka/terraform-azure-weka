#!/bin/bash

# Usage:
# write_function_hash_to_variables.sh <os_name> <function_code_path>

os_name="$1"
function_code_path="$2"

new_function_app_zip_version=$(./zip_function_app_creation/get_function_app_hash.sh ${os_name} ${function_code_path})
old_function_app_zip_version=$(awk '/Function app code version/{getline;print $NF;}' variables.tf | tr -d \")

echo "Replacing '$old_function_app_zip_version' function_app_version to '$new_function_app_zip_version'"
if [ $os_name == "darwin" ]; then
    sed -i '' "s/$old_function_app_zip_version/$new_function_app_zip_version/" variables.tf
else
    sed -i "s/$old_function_app_zip_version/$new_function_app_zip_version/" variables.tf
fi
