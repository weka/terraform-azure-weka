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

# Update nodepool to have multi nics
interfaces=$(az vmss show --resource-group $aks_rg_name --name $aks_vmss_name --query "virtualMachineProfile.networkProfile.networkInterfaceConfigurations")

interfaces_updated="$interfaces"

updates_network_profile=$(echo $interfaces | jq ".[]" )

# Loop through each update
for  (( i=2; i <= $nics; i++ )); do
    interface="$updates_network_profile"
    interface=$(echo "$interface" | jq ".primary = false")
    interface=$(echo "$interface" | jq ".ipConfigurations[0].name = \"ipconfig$i\"")
    interface=$(echo "$interface" | jq ".name = \"ipconfig$i\"")
    echo "$interface"
    interfaces_updated=$(echo "$interfaces_updated" | jq ". + [$interface]")
done

az vmss update --resource-group $aks_rg_name --name $aks_vmss_name --set "virtualMachineProfile.networkProfile.networkInterfaceConfigurations=$interfaces_updated"

# Set aks credentials
az aks get-credentials --resource-group $rg_name --name $aks_cluster_name --overwrite-existing

# Config kube yaml
weka_password=$(az keyvault secret show --vault-name $vault_name --name weka-password | jq -r .value)
weka_password_base64=$(echo $weka_password | base64)
backend_ips=$(az vmss nic list -g $rg_name --vmss-name $backend_vmss_name --subscription $subscription_id --query "[].ipConfigurations[]" | jq -r '.[] | select(.name=="ipconfig0")'.privateIPAddress)
endpoint=()

while IFS= read -r ip; do
  backend_ip=$ip
  endpoint+="$ip:14000,"
done <<< "$backend_ips"

endpoint=$${endpoint%,}
endpoint_base64=$(echo $endpoint | base64)

sed -i "" "/^\([[:space:]]*password: \).*/s//\1$${weka_password_base64}/" $yamls_path/yamls/secret.yaml
sed -i "" "/^\([[:space:]]*endpoints: \).*/s//\1$${endpoint_base64}/" $yamls_path/yamls/secret.yaml

yq eval --inplace ".spec.template.spec.containers[].env[] |= select(.name == \"NICS\").value = \"$nics\"" $yamls_path/yamls/daemonset.yaml
yq eval --inplace ".spec.template.spec.containers[].env[] |= select(.name == \"BACKEND_IP\").value = \"$backend_ip\"" $yamls_path/yamls/daemonset.yaml
yq eval --inplace ".spec.template.spec.containers[].env[] |= select(.name == \"FRONTEND_CONTAINER_CORES_NUM\").value = \"$frontend_container_cores_num\"" $yamls_path/yamls/daemonset.yaml


kubectl apply -f $yamls_path/yamls/configmap.yaml
kubectl apply -f $yamls_path/yamls/daemonset.yaml

#scale up nodepool
az vmss scale --new-capacity $node_count --resource-group $aks_rg_name --name $aks_vmss_name

#Install csi plugin
#helm repo add csi-wekafs https://weka.github.io/csi-wekafs
#helm install csi-wekafs csi-wekafs/csi-wekafsplugin --namespace csi-wekafs --create-namespace
#
#kubectl apply -f $yamls_path/yamls/secret.yaml
#kubectl apply -f $yamls_path/yamls/storageclass.yaml
#kubectl apply -f $yamls_path/yamls/deployment.yaml
