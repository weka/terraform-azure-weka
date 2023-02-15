import pytest
from qa.helpers.deploy import extract_private_key_path, extract_instance_ip
from qa.helpers.ssh_action import get_weka_status_on_instance


# Test suite description
# Precondition:
# 1). Add variables: protection_level, stripe_width, hotspare to tf variables in example\public_network
# 2). Terraform init
# 3). Terraform deploy
# 4). Waiting for the cluster
# Steps:
# 1). Get instance0 ip address
# 2). Connect to this instance via ssh and run 'weka status'
# 3). Get
# 4). Validate this values
# Postcondition:
# 1). Terraform destroy


@pytest.mark.regression
@pytest.mark.parametrize('data_protection_args', [('Default values', 'protection: 3+2', 'hot spare: 1'),
                                                  ({"cluster_size": 10, "protection_level": 4, "hotspare": 1},
                                                   'protection: 7+2', 'hot spare: 1'),
                                                  ({"cluster_size": 19, "protection_level": 2, "hotspare": 1},
                                                   'protection: 16+2', 'hot spare: 1')])
def test_data_protection_with_default_values(command_line_args, deploy_weka, data_protection_args):
    deploy_stdout = deploy_weka
    p_key = extract_private_key_path(deploy_stdout)
    instance_ip = extract_instance_ip(prefix=command_line_args.get('prefix'),
                                      clustername=command_line_args.get('cluster_name'),
                                      id=0, deploy_stdout=deploy_stdout)
    status_stdout = get_weka_status_on_instance(ip_address=instance_ip, key_file_path=p_key)
    protection_check, hot_spare_check = False, False
    for line in status_stdout:
        if line == data_protection_args[1]:
            protection_check = True
        if line.startswith(data_protection_args[2]):
            hot_spare_check = True
    assert protection_check and hot_spare_check
