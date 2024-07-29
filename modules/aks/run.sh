set -ex

rg_name=$1
aks_cluster_name=$2
vault_name=$3
backend_vmss_name=$4
subscription_id=$5
nics=$6
nodepool_number=$7
frontend_container_cores_num=$8
yamls_path=$9
ml_name=$10

apt install yq -y || brew install yq || yum install yq -y || true
apt install jq -y || brew install jq || yum install jq -y || true

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

endpoint=${endpoint%,}
endpoint_base64=$(echo $endpoint | base64)

sed -i "" "/^\([[:space:]]*password: \).*/s//\1${weka_password_base64}/" ${yamls_path}/yamls/secret.yaml
sed -i "" "/^\([[:space:]]*endpoints: \).*/s//\1${endpoint_base64}/" ${yamls_path}/yamls/secret.yaml

yq eval --inplace ".spec.template.spec.containers[].env[] |= select(.name == \"NICS\").value = \"${nics}\"" ${yamls_path}/yamls/daemonset.yaml
yq eval --inplace ".spec.template.spec.containers[].env[] |= select(.name == \"BACKEND_IP\").value = \"${backend_ip}\"" ${yamls_path}/yamls/daemonset.yaml
yq eval --inplace ".spec.template.spec.containers[].env[] |= select(.name == \"FRONTEND_CONTAINER_CORES_NUM\").value = \"${frontend_container_cores_num}\"" ${yamls_path}/yamls/daemonset.yaml


kubectl apply -f ${yamls_path}/yamls/configmap.yaml
kubectl apply -f ${yamls_path}/yamls/daemonset.yaml

#scale up nodepool
az vmss scale --new-capacity $nodepool_number --resource-group $aks_rg_name --name $aks_vmss_name

#Install csi plugin
helm repo add csi-wekafs https://weka.github.io/csi-wekafs
helm install csi-wekafs csi-wekafs/csi-wekafsplugin --namespace csi-wekafs --create-namespace

kubectl apply -f ${yamls_path}/yamls/secret.yaml
kubectl apply -f ${yamls_path}/yamls/storageclass.yaml
kubectl apply -f ${yamls_path}/yamls/deployment.yaml

#aks_id=$(az aks show --name $aks_cluster_name --resource-group $rg_name | jq ".id")
#region=$(az aks show --name $aks_cluster_name --resource-group $rg_name | jq -r ".location")

#TOKEN=$(az account get-access-token --query accessToken -o tsv)
#az rest --method GET --url "https://$region.api.azureml.ms/rp/workspaces/subscriptions/$subscription_id/resourceGroups/$rg_name/providers/Microsoft.MachineLearningServices/workspaces/$ml_name/outboundNetworkDependenciesEndpoints?api-version=2018-03-01-preview" --header Authorization="Bearer $TOKEN"

#az k8s-extension create --name $ml_name --extension-type Microsoft.AzureML.Kubernetes \
#--config nodeSelector.agentpool=clients enableTraining=True enableInference=True inferenceRouterServiceType=LoadBalancer allowInsecureConnections=True InferenceRouterHA=False \
#--cluster-type managedClusters --cluster-name $aks_cluster_name --resource-group $rg_name --scope cluster --debug


#az ml compute attach --resource-group $rg_name --workspace-name $ml_name --type Kubernetes --name k8s-compute --resource-id $aks_id --identity-type SystemAssigned --no-wait

#kubectl apply -f ${yamls_path}/yamls/statefulset.yaml
#kubectl apply -f ${yamls_path}/yamls/ml-pvs.yaml
