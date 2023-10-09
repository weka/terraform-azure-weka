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
  local  __result=$1
  response=$(curl https://$app_name.azurewebsites.net/api/status\?code\=$function_key --no-progress-meter)
  actual_total=$(echo "$response" | jq -r '.weka_status.drives.total')
  actual_active=$(echo "$response" | jq -r '.weka_status.drives.active')
  actual_clusterized=$(echo "$response" | jq -r '.clusterized')
  if [ "$actual_clusterized" != true ]; then
    echo "Weka clusterization didn't finish"
    eval "$__result"=false
  elif [ "$actual_total" != "$expected_capacity" ]; then
    echo "Weka drive containers total capacity isn't satisfied. actual:$actual_total  expected:$expected_capacity"
    eval "$__result"=false
  elif [ "$actual_active" != "$expected_capacity" ]; then
    echo "Weka drive containers active capacity isn't satisfied. actual:$actual_active  expected:$expected_capacity"
    eval "$__result"=false
  else
    eval "$__result"=true
  fi
}

count=1
sleep 60
validate_status last_status
while [ "$last_status" != true ] && [ $count -lt "$timeout" ]
do
  echo "going to sleep for 1 minute ($count out of $timeout minutes passed)"
  sleep 60
  count=$((count + 1 ))
  validate_status last_status
done

if [ "$last_status" == true ]; then
  echo "weka cluster reached expected state after $count minutes!"
else
  echo "weka cluster didn't reach expected state during $timeout minutes!"
  exit 1
fi
