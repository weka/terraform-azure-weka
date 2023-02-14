import pytest
from qa.helpers.deploy import *
from qa.helpers.file_manipulation import extend_tf_variables


@pytest.fixture(scope='session')
def add_data_protection_variables():
    extend_tf_variables('public_network', 'additional_data_protection_variables.tf')

@pytest.fixture(scope='function')
def deploy_weka(command_line_args, add_data_protection_variables, data_protection_args):
    params = {}
    params.update(command_line_args)
    if type(data_protection_args[0]) == dict:
        params.update(data_protection_args[0])
    output = deploy_env('public_network', **params)
    # Waiting for the cluster
    key = get_function_key(prefix=command_line_args.get('prefix'),
                           cluster_name=command_line_args.get('cluster_name'),
                           rg_name=command_line_args.get('rg_name'),
                           subscription_id=command_line_args.get('subscription_id'))
    waiting_for_the_cluster(prefix=command_line_args.get('prefix'),
                            cluster_name=command_line_args.get('cluster_name'),
                            key=key)
    yield output
    destroy_env('public_network', **command_line_args)
