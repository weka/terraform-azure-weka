#!/bin/bash

prefix="$1"
cluster_name="$2"
rg_name="$3"
subscription_id="$4"
new_capacity="$5"

# Get the function key
app_name="${prefix}-${cluster_name}-function-app"
function_key=$(az functionapp keys list --name $app_name --resource-group $rg_name --subscription $subscription_id --query functionKeys -o tsv)

# resize cluster
curl "https://$app_name.azurewebsites.net/api/resize?code=$function_key" -H \"Content-Type:application/json\" -d '{"value":'"$new_capacity"'}'
