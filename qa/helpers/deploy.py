import requests
import time
import re
from qa.helpers.core import ShellClient, BASE_DIR, logger


def deploy_env(example_name='public_network', **kwargs):
    # terraform init
    sc = ShellClient()
    sc.execute_command(f'cd {BASE_DIR}/examples/{example_name} ; terraform init')
    # terraform apply
    attributes = ''.join([f' -var="{k}={v}"' for k, v in kwargs.items()])
    cmd = f'cd {BASE_DIR}/examples/{example_name} ; terraform apply -var-file vars.auto.tfvars{attributes} ' \
          f'-auto-approve'
    s = ShellClient()
    s.execute_command(cmd)
    return s.stdout


def get_function_key(prefix, cluster_name, rg_name, subscription_id):
    s = ShellClient()
    s.execute_command(
        f'az functionapp keys list --name {prefix}-{cluster_name}-function-app --resource-group {rg_name} '
        f'--subscription {subscription_id} --query functionKeys -o tsv')
    return s.stdout[0]


def waiting_for_the_cluster(prefix, cluster_name, key, operation_timeout=1800):
    timeout = time.time() + operation_timeout
    while time.time() < timeout:
        r = requests.get(url=f'https://{prefix}-{cluster_name}-function-app.azurewebsites.net/api/status',
                         params={"code": key})
        assert r.status_code == 200
        if r.json()["clusterized"]:
            logger.info('Weka cluster is ready!')
            break
        else:
            logger.info("Weka cluster isn't ready yet, going to sleep for 60 s")
            time.sleep(60)


def extract_instance_ip(prefix, clustername, id, deploy_stdout):
    ip_address_pattern = re.compile(r'(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})')
    for item in reversed(deploy_stdout):
        if item.startswith(f'"{prefix}-{clustername}-backend-{id}'):
            return ip_address_pattern.search(item)[0]


def extract_private_key_path(deploy_stdout):
    for item in reversed(deploy_stdout):
        if item.startswith('SSH-KEY-PATH'):
            keys = item.replace('"', '').split()
            return keys[-1]


def destroy_env(example_name='public_network', **kwargs):
    attributes = ''.join([f' -var="{k}={v}"' for k, v in kwargs.items()])
    cmd = f'cd {BASE_DIR}/examples/{example_name} ; terraform destroy -var-file vars.auto.tfvars{attributes} ' \
          f'-auto-approve'
    s = ShellClient()
    s.execute_command(cmd)
