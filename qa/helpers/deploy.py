import time
import json
import uuid
from qa.helpers.core import ShellClient, BASE_DIR, logger
from qa.helpers.endpoints import get_cluster_status


class TerraformAction:

    def __init__(self, worker_id, client_id, client_secret, tenant_id, get_weka_io_token, subscription_id):
        self.__sc = ShellClient()
        self.__client_id = client_id
        self.__client_secret = client_secret
        self.__tenant_id = tenant_id
        self.__get_weka_io_token = get_weka_io_token
        self.__subscription_id = subscription_id
        self.worker_id = worker_id
        self.rg = f'rg-{uuid.uuid4()}'
        self.cluster_name = f'{worker_id}{str(uuid.uuid4())[:5]}'
        self.prefix = 'at'
        self.working_directory_path = f'{BASE_DIR}/qa/working_directory/{self.worker_id}'
        self.__create_working_directory()

    def __create_working_directory(self):
        self.__sc.execute_command(f'mkdir {self.working_directory_path}')

    def create_tf_configuration_file(self, **kwargs):
        variables = {
            "get_weka_io_token": self.__get_weka_io_token,
            "client_id": self.__client_id,
            "tenant_id": self.__tenant_id,
            "client_secret": self.__client_secret,
            "subscription_id": self.__subscription_id,
            "address_space": "10.0.0.0/16",
            "cluster_name": self.cluster_name,
            "cluster_size": 6,
            "instance_type": "Standard_L8s_v3",
            "location": "eastus",
            "prefix": self.prefix,
            "rg_name": self.rg,
            "set_obs_integration": True,
            "subnet_prefixes": ["10.0.1.0/24"],
            "subnet_delegation": "10.0.2.0/25",
            "tiering_ssd_percent": 20}
        variables.update(kwargs)
        setattr(self, 'cluster_size', variables['cluster_size'])
        with open(f"{BASE_DIR}/qa/test_data/template.tf", "r") as template_file:
            template = template_file.read()
        with open(f"{self.working_directory_path}/main.tf", "w") as output_file:
            output_file.write(template)
            output_file.write('\n\n# Create necessary variables\nlocals {')
            for k, v in variables.items():
                value = json.dumps(v)
                output_file.write(f'\n  {k} = {value}')
            output_file.write("\n}\n")

    def apply(self):
        cmd = f'cd {self.working_directory_path} ; terraform init ; terraform apply -auto-approve'
        self.__sc.execute_command(cmd)

    def destroy(self):
        cmd = f'cd {self.working_directory_path} ; terraform destroy -auto-approve'
        self.__sc.execute_command(cmd)

    def delete_working_dir(self):
        self.__sc.execute_command(f'rm -rf {self.working_directory_path}')

    def check_cluster_status(self, key, expected_capacity):
        result = False
        response = get_cluster_status(self.prefix, self.cluster_name, key)
        assert response.status_code == 200, f"Unexpected error code from GET /status: {response.status_code}"
        actual_is_clusterized = response.json()["clusterized"]
        actual_total = response.json()['weka_status']['drives']['total']
        actual_active = response.json()['weka_status']['drives']['active']
        if not actual_is_clusterized:
            logger.info("Weka clusterization didn't finish!")
        elif actual_total != expected_capacity:
            logger.info(f"Weka drive containers total capacity isn't satisfied. actual: {actual_total}  expected: "
                        f"{expected_capacity}")
        elif actual_active != expected_capacity:
            logger.info(f"Weka drive containers active capacity isn't satisfied. actual: {actual_active}  expected: "
                        f"{expected_capacity}")
        else:
            result = True
        return result

    def waiting_for_the_cluster(self, key, expected_capacity, operation_timeout=1800):
        # operation timeout should be in seconds
        timeout = time.time() + operation_timeout
        count = 0
        while time.time() < timeout:
            result = self.check_cluster_status(key, expected_capacity)
            if result:
                logger.info(f"Weka cluster reached expected state after {count} minutes!")
                return True
            else:
                logger.info(f"going to sleep for 1 minute ({count} out of {int(operation_timeout/60)} minutes passed)")
                time.sleep(60)
                count += 1
        logger.info(f"Weka cluster didn't reach expected state during {int(operation_timeout/60)} minutes!")
