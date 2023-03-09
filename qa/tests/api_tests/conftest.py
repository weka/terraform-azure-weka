import dataclasses

import pytest
from contextlib import contextmanager
from qa.helpers.deploy import WekaTF
from qa.helpers.azure_helper import AzureHelper
from qa.helpers.core import logger, CloudHelper
from dataclasses import dataclass
from typing import ContextManager


@dataclass
class TFDeployment:
    tf: WekaTF
    key: str
    cloud_helper: CloudHelper

@contextmanager
def setup_env(command_line_args, worker_id, **kwargs) -> ContextManager[TFDeployment]:
    if command_line_args.get('cloud') == 'Azure':
        cloud_helper = AzureHelper(**command_line_args)
    else:
        raise NotImplementedError("only azure supported atm")
    tf = WekaTF(worker_id, **command_line_args)
    tf.create_tf_configuration_file(**kwargs)
    try:
        tf.apply()
        key = cloud_helper.get_function_key(rg=tf.rg, prefix=tf.prefix, cluster_name=tf.cluster_name)
        tf.waiting_for_the_cluster(key, tf.cluster_size)
        yield TFDeployment(tf=tf, key=key, cloud_helper=cloud_helper)
    except Exception as deploy_exception:
        logger.error(f'Deploy is failed. Exception occurs: {deploy_exception}!')
        try:
            tf.destroy()
        except Exception as destroy_exception:
            logger.error(f'Destroy is failed. Exception occurs: {destroy_exception}!')
        finally:
            cloud_helper.delete_resource_group(tf.rg)
            tf.delete_working_dir()
        raise deploy_exception


def destroy_env(tf):
    tf.destroy()
    tf.delete_working_dir()


@pytest.fixture(scope='module')
def deploy_env_module(command_line_args, worker_id):
    with setup_env(command_line_args, worker_id) as result:
        yield result
    destroy_env(result.tf)


@pytest.fixture(scope='function')
def deploy_env_function(command_line_args, worker_id):
    with setup_env(command_line_args, worker_id) as result:
        yield result
    destroy_env(result.tf)


@pytest.fixture(scope='function')
def deploy_env_with_data_protection_values(command_line_args, worker_id, data_protection_args):
    kwargs = data_protection_args[0]
    with setup_env(command_line_args, worker_id, **kwargs) as result:
        yield result
    destroy_env(result.tf)
