#!/bin/bash

# Usage:
# create_function_binarie.sh <os_name> <function_code_path> <function_binaries_dir> 

set -e

os_name="$1"
function_code_path="$2"
function_binaries_dir="$3"

current_script_dir=$(dirname ${BASH_SOURCE[0]})

function_app_code_hash="$($current_script_dir/get_function_app_hash.sh ${os_name} ${function_code_path})"
echo "function_app_code_hash: $function_app_code_hash"

function_binarie_dir="${function_binaries_dir}/${function_app_code_hash}/"

echo "Building function code..."

echo "Creating dir $function_binarie_dir"
mkdir -p $function_binarie_dir

# Go to the function code directory
cd $function_code_path

# Build the function app
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $function_binarie_dir

echo "Function code built."
