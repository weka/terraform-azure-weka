#!/bin/bash

# Usage:
# write_code_hash_to_variables.sh <os_name> <function_code_path>

os_name="$1"
function_code_path="$2"

current_script_dir=$(dirname ${BASH_SOURCE[0]})

new_code_version=$(${current_script_dir}/get_code_hash.sh ${os_name} ${function_code_path})
old_code_version=$(awk '/Function app code version/{getline;print $NF;}' ${current_script_dir}/../variables.tf | tr -d \")

echo "Replacing '$old_code_version' function_app_version to '$new_code_version'"
if [ $os_name == "darwin" ]; then
    sed -i '' "s/$old_code_version/$new_code_version/" ${current_script_dir}/../variables.tf
else
    sed -i "s/$old_code_version/$new_code_version/" ${current_script_dir}/../variables.tf
fi
