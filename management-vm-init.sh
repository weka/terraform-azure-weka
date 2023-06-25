#!/bin/bash
set -ex

export AZURE_CLIENT_ID="${azure_client_id}"
export AZURE_CLIENT_SECRET="${azure_client_secret}"
export AZURE_TENANT_ID="${azure_tenant_id}"

export STATE_STORAGE_NAME="${state_storage_name}"
export STATE_CONTAINER_NAME="${state_container_name}"
export HOSTS_NUM="${hosts_num}"
export CLUSTER_NAME="${cluster_name}"
export PROTECTION_LEVEL="${protection_level}"
export STRIPE_WIDTH="${stripe_width}"
export HOTSPARE="${hotspare}"
export VM_USERNAME="${vm_username}"
export COMPUTE_MEMORY="${compute_memory}"
export SUBSCRIPTION_ID="${subscription_id}"
export RESOURCE_GROUP_NAME="${resource_group_name}"
export LOCATION="${location}"
export SET_OBS="${set_obs}"
export OBS_NAME="${obs_name}"
export OBS_CONTAINER_NAME="${obs_container_name}"
export OBS_ACCESS_KEY="${obs_access_key}"
export NUM_DRIVE_CONTAINERS="${num_drive_containers}"
export NUM_COMPUTE_CONTAINERS="${num_compute_containers}"
export NUM_FRONTEND_CONTAINERS="${num_frontend_containers}"
export NVMES_NUM="${nvmes_num}"
export TIERING_SSD_PERCENT="${tiering_ssd_percent}"
export PREFIX="${prefix}"
export KEY_VAULT_URI="${key_vault_uri}"
export INSTANCE_TYPE="${instance_type}"
export INSTALL_DPDK="${install_dpdk}"
export NICS_NUM="${nics_num}"
export INSTALL_URL='${install_url}'  # ensure raw string
export LOG_LEVEL="${log_level}"
export SUBNET="${subnet}"
export HTTP_SERVER_HOST="$(ip route get 1 | awk '{print $(NF-2);exit}')"
export HTTP_SERVER_PORT="${http_server_port}"

echo "Getting function app binary"
curl -o weka-deployment "${function_app_code_url}"
chmod +x weka-deployment 
./weka-deployment &
