#!/bin/bash
# Usage:
# get_function_app_hash.sh <os_name> <function_code_path>

set -e

os_name="$1"
function_code_path="$2"

# make sure go.mod and go.sum are up-to-date
cd "$function_code_path"
rm -rf weka-deployment # remove local build in case it exists
go mod tidy > /dev/null 2>&1

if [ $os_name == "darwin" ]; then
    function_app_code_hash="$(find ${function_code_path} -type f | LC_ALL=C sort | xargs -n1 md5 | awk {'print $NF'} ORS='' | md5)"
else
    function_app_code_hash="$(find ${function_code_path} -type f | LC_ALL=C sort | xargs -n1 md5sum | awk '{print $1}' ORS='' | md5sum | awk '{print $1}')"
fi
echo $function_app_code_hash
