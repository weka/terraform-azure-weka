#!/bin/bash

prefix="$1"
cluster_name="$2"
rg_name="$3"
subscription_id="$4"
timeout="$5"

# Get the function key
app_name="${prefix}-${cluster_name}-function-app"
function_key=$(az functionapp keys list --name $app_name --resource-group $rg_name --subscription $subscription_id --query functionKeys -o tsv)

# get status
result=$(curl https://$app_name.azurewebsites.net/api/status\?code\=$function_key | jq -r '.clusterized')
count=1
while [ "$result" != true ] && [ $count -le "$timeout" ]
do
  echo "weka cluster isn't ready yet, going to sleep for 60 s"
  sleep 60
  result=$(curl https://$app_name.azurewebsites.net/api/status\?code\=$function_key | jq -r '.clusterized')
  count=$(( $count + 1 ))
done

if [ $count -gt "$timeout" ]; then
  echo "weka cluster wasn't ready during $timeout minutes!"
	exit 1
fi

