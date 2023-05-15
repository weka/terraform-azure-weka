#!/bin/bash

# Usage:
# build_function_code.sh <os_name> <function_code_path> <function_zip_dir> 

set -e

os_name="$1"
function_code_path="$2"
function_zip_dir="$3"
function_triggers_path="$(dirname "$function_code_path")/triggers"

current_script_dir=$(dirname ${BASH_SOURCE[0]})

function_app_code_hash="$($current_script_dir/get_function_app_hash.sh ${os_name} ${function_code_path})"
echo "function_app_code_hash: $function_app_code_hash"

function_zip_path="${function_zip_dir}/${function_app_code_hash}.zip"

echo "Building function code..."

echo "function_zip_path: $function_zip_path"
func_zip_dir="$(dirname $function_zip_path)"
echo "Creating dir $func_zip_dir"
mkdir -p $func_zip_dir

# Go to the function code directory
cd $function_code_path

# Build the function app
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $function_triggers_path

echo "Function code built."

echo "Creating zip archive..."

# Create the zip archive
old_dir=$(pwd)
cd $function_triggers_path
zip -r $function_zip_path *
cd $old_dir

echo "Zip archive created: $function_zip_path"
