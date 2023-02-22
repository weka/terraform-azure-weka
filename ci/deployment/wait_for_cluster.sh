#!/bin/bash

prefix="$1"
cluster_name="$2"
rg_name="$3"
subscription_id="$4"
expected_capacity="$5"
timeout="$6" # in minutes

# Get the function key
app_name="${prefix}-${cluster_name}-function-app"
function_key=$(az functionapp keys list --name $app_name --resource-group $rg_name --subscription $subscription_id --query functionKeys -o tsv)


function validate_status () {
  response=$(curl https://$app_name.azurewebsites.net/api/status\?code\=$function_key --no-progress-meter)
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
sleep 60
last_status="$(validate_status)"
while [ "$last_status" != true ] && [ $count -lt "$timeout" ]
do
  echo "weka cluster didn't reach expected state after $count minutes, going to sleep for 1 minute"
  sleep 60
  count=$((count + 1 ))
  last_status="$(validate_status)"
done

if [ "$last_status" == true ]; then
  echo "weka cluster reached expected state after $count minutes!"
else
  echo "weka cluster didn't reach expected state during $timeout minutes!"
  exit 1
fi

