#!/bin/bash

# Usage:
# create_binary.sh <os_name> <function_code_path> <binaries_dir> 

set -e

os_name="$1"
function_code_path="$2"
binaries_dir="$3"

current_script_dir=$(dirname ${BASH_SOURCE[0]})

function_app_code_hash="$($current_script_dir/get_code_hash.sh ${os_name} ${function_code_path})"
echo "function_app_code_hash: $function_app_code_hash"

binary_dir="${binaries_dir}/${function_app_code_hash}/"

echo "Building function code..."

echo "Creating dir $binary_dir"
mkdir -p $binary_dir

# Go to the function code directory
cd $function_code_path

# Build the function app
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $binary_dir

cd $current_script_dir
echo "Function code built."
