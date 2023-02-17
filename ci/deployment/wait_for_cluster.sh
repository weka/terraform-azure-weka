#!/bin/bash

prefix="$1"
cluster_name="$2"
rg_name="$3"
subscription_id="$4"
expected_capacity="$5"
timeout="$6"

# Get the function key
app_name="${prefix}-${cluster_name}-function-app"
function_key=$(az functionapp keys list --name $app_name --resource-group $rg_name --subscription $subscription_id --query functionKeys -o tsv)


function validate_status () {
  response=$(curl https://$app_name.azurewebsites.net/api/status\?code\=$function_key)
  actual_total=$(echo $response | jq -r '.weka_status.drives.total')
  actual_active=$(echo $response | jq -r '.weka_status.drives.active')
  actual_clusterized=$(echo $response | jq -r '.clusterized')
  if [ "$actual_total" = $expected_capacity ] && [ "$actual_active" = $expected_capacity ] && [ "$actual_clusterized" = true ]
  then
    echo true
  else
    echo false
  fi
}

count=1

while [ "$(validate_status)" != true ] && [ $count -le "$timeout" ]
do
  echo "weka cluster isn't ready yet, going to sleep for 60 s"
  sleep 60
  validate_status
  count=$((count + 1 ))
done
if [ $count -gt "$timeout" ]; then
  echo "weka cluster wasn't ready during $timeout minutes!"
	exit 1
fi

