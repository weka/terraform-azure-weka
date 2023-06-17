#!/bin/bash

# Usage:
# get_code_hash.sh <os_name> <code_path>

os_name="$1"
code_path="$2"

# make sure go.mod and go.sum are up-to-date
go mod tidy > /dev/null 2>&1

if [ $os_name == "darwin" ]; then
    code_hash="$(find ${code_path} -type f | LC_ALL=C sort | xargs -n1 md5 | awk {'print $NF'} ORS='' | md5)"
else
    code_hash="$(find ${code_path} -type f | LC_ALL=C sort | xargs -n1 md5sum | awk '{print $1}' ORS='' | md5sum | awk '{print $1}')"
fi
echo $code_hash
