# get token for key vault access
access_token=$(curl 'http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https%3A%2F%2Fvault.azure.net' -H Metadata:true | jq -r '.access_token')
# get key vault secret
function_app_key=$(curl "${key_vault_url}secrets/${vault_function_app_key_name}?api-version=2016-10-01" -H "Authorization: Bearer $access_token" | jq -r '.value')

function report {
  local json_data=$1
  curl ${report_function_url}?code="$function_app_key" -H 'Content-Type:application/json' -d "$json_data"
}


# wait for all containers to be ready
cluster_size="${gateways_number}"
max_retries=60
for (( retry=1; retry<=max_retries; retry++ )); do
    # get all UP gateway container ids
    all_container_ids=$(weka cluster container | grep frontend0 | grep ${gateways_name} | grep UP | awk '{print $1}')
    # if number of all_container_ids < cluster_size, do nothing
    all_container_ids_number=$(echo "$all_container_ids" | wc -l)
    if (( all_container_ids_number < cluster_size )); then
        echo "$(date -u): not all containers are ready - do retry $retry of $max_retries"
        sleep 20
    else
        echo "$(date -u): all containers are ready"
        break
    fi
done

if (( retry > max_retries )); then
    err_msg="timeout: not all containers are ready after $max_retries attempts."
    echo "$(date -u): $err_msg"
    report "{\"hostname\": \"$HOSTNAME\", \"type\": \"error\", \"message\": \"$err_msg\"}"
    exit 1
fi

echo "$(date -u): Done running validation for protocol"
