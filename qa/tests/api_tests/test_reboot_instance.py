import pytest
import time
from qa.helpers.endpoints import get_cluster_status
from qa.helpers.core import logger


@pytest.mark.regression
def test_reboot_instance(deploy_env_function):
    # TEST DESCRIPTION:
    # Create default infrastructure with 6 vms
    # Wait for clusterization
    # Start restart command for one vm in the scale set (using Azure SDK)
    # Wait and verify that active drivers capacity becomes 5
    # Wait for starting instance after reboot and verify active drivers capacity becomes 6 and cluster is healthy
    # ====
    # define operation timeout in secludes
    OPERATION_TIMEOUT = 600
    tf, key, cloud_helper = deploy_env_function
    cloud_helper.reboot_instance(tf.rg, tf.prefix, tf.cluster_name)
    # Wait and verify that active drivers capacity becomes 5
    timeout = time.time() + OPERATION_TIMEOUT
    actual_active = None
    while time.time() < timeout:
        response = get_cluster_status(tf.prefix, tf.cluster_name, key)
        actual_active = response.json()['weka_status']['drives']['active']
        if actual_active == 5:
            break
    if actual_active != 5:
        raise TimeoutError(f'During {OPERATION_TIMEOUT} cluster active capacity is not expected. Expected = 5, '
                           f'Actual = {actual_active}')
    # Wait for starting instance after reboot and verify active drivers capacity becomes 6 and cluster is healthy
    assert tf.waiting_for_the_cluster(key, 6, operation_timeout=OPERATION_TIMEOUT)



