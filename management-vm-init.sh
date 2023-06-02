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
export SUBNETS="${subnets}"
export HTTP_SERVER_HOST="$(ip route get 1 | awk '{print $(NF-2);exit}')"
export HTTP_SERVER_PORT="${http_server_port}"

echo "Getting function app binary"
curl -o weka-deployment "${function_app_code_url}"
chmod +x weka-deployment 
./weka-deployment &

cat <<'EOF' > run_scale_down_workflow.py
import requests
import sys

url = sys.argv[1]

class FunctionCallException(Exception):
    def __init__(self, message, code):
        self.message = message
        self.code = code
        super().__init__(message)
    
    def __str__(self):
        return f"{self.__class__.__name__} - status_code: {self.code}, response: {self.message}"

def call_api_function(name, json_data):
    headers = {'Content-Type': 'application/json'}
    response = requests.post(f"{url}/{name}", headers=headers, data=json_data)
    if 200 <= response.status_code < 300:
        return response.text
    raise FunctionCallException(response.text, response.status_code)


def run_scale_down_flow():
    fetch_resp = call_api_function('fetch', None)
    scale_down_resp = call_api_function('scale_down', fetch_resp)
    terminate_resp = call_api_function('terminate', scale_down_resp)
    transient_resp = call_api_function('transient', terminate_resp)
    return transient_resp

print('Running scale_down_workflow')
try:
    resp = run_scale_down_flow()
    print(resp)
except FunctionCallException as e:
    print(str(e))
EOF

touch run_scale_functions
echo 'MAILTO=""' >> run_scale_functions
# add commands to new cron file
echo "*/1 * * * * curl -s --show-error -X POST $(ip route get 1 | awk '{print $(NF-2);exit}'):${http_server_port}/scale_up 2>&1 | /usr/bin/logger -t scale_functions" >> run_scale_functions
echo "*/1 * * * * /usr/bin/python3 $(pwd)/run_scale_down_workflow.py http://$(ip route get 1 | awk '{print $(NF-2);exit}'):${http_server_port} 2>&1 | /usr/bin/logger -t scale_functions" >> run_scale_functions
# install new cron file
crontab run_scale_functions
rm run_scale_functions
