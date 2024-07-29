set -ex

rg_name=${rg_name}
aks_cluster_name=${aks_cluster_name}
vault_name=${key_vault_name}
backend_vmss_name=${backend_vmss_name}
subscription_id=${subscription_id}
nics=${nics}
node_count=${node_count}
frontend_container_cores_num=${frontend_container_cores_num}
yamls_path=${yamls_path}

# install yq if not installed
if ! command -v yq &> /dev/null
then
    echo "yq could not be found"
    apt install yq -y || brew install yq || yum install yq -y || true
fi

# install jq if not installed
if ! command -v jq &> /dev/null
then
    echo "jq could not be found"
    apt install jq -y || brew install jq || yum install jq -y || true
fi

aks_rg_name=$(az aks show -n $aks_cluster_name -g $rg_name | jq -r ".nodeResourceGroup")
aks_vmss_name=$(az vmss list -g $aks_rg_name | jq -r ".[].name" | grep clients)

# Set aks credentials
az aks get-credentials --resource-group $rg_name --name $aks_cluster_name --overwrite-existing

# Config kube yaml
backend_ips=$(az vmss nic list -g $rg_name --vmss-name $backend_vmss_name --subscription $subscription_id --query "[].ipConfigurations[]" | jq -r '.[] | select(.name=="ipconfig0")'.privateIPAddress)
backend_ip=$(echo "$backend_ips" | head -n 1)

yq eval --inplace ".spec.template.spec.containers[].env[] |= select(.name == \"NICS\").value = \"$nics\"" $yamls_path/yamls/daemonset.yaml
yq eval --inplace ".spec.template.spec.containers[].env[] |= select(.name == \"BACKEND_IP\").value = \"$backend_ip\"" $yamls_path/yamls/daemonset.yaml
yq eval --inplace ".spec.template.spec.containers[].env[] |= select(.name == \"FRONTEND_CONTAINER_CORES_NUM\").value = \"$frontend_container_cores_num\"" $yamls_path/yamls/daemonset.yaml

kubectl apply -f $yamls_path/yamls/configmap.yaml
kubectl apply -f $yamls_path/yamls/daemonset.yaml

#scale up nodepool
az vmss scale --new-capacity $node_count --resource-group $aks_rg_name --name $aks_vmss_name
