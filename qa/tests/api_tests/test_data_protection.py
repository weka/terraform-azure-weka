import pytest
from qa.helpers.endpoints import get_cluster_status


@pytest.mark.regression
def test_data_protection_with_default_values(deploy_env_function):
    tf, key = deploy_env_function
    response = get_cluster_status(tf.prefix, tf.cluster_name, key)
    assert response.json()['weka_status']['hot_spare'] == 1, "Unexpected hot_spare value"
    assert response.json()['weka_status']["stripe_data_drives"] == 3, "Unexpected stripe_data_drives value"
    assert response.json()['weka_status']["stripe_protection_drives"] == 2, "Unexpected stripe_protection_drives value"


@pytest.mark.regression
@pytest.mark.parametrize('data_protection_args',
                         [({"cluster_size": 10, "protection_level": 4, "hotspare": 1}, (7, 2, 1)),
                          ({"cluster_size": 19, "protection_level": 2, "hotspare": 1}, (16, 2, 1))])
def test_data_protection_with_custom_values(deploy_env_with_data_protection_values, data_protection_args):
    tf, key = deploy_env_with_data_protection_values
    response = get_cluster_status(tf.prefix, tf.cluster_name, key)
    expected_stripe_data_drives, expected_stripe_protection_drives, expected_hot_spare = data_protection_args[1]
    assert response.json()['weka_status']['hot_spare'] == expected_hot_spare, \
        "Unexpected hot_spare value"
    assert response.json()['weka_status']["stripe_data_drives"] == expected_stripe_data_drives, \
        "Unexpected stripe_data_drives value"
    assert response.json()['weka_status']["stripe_protection_drives"] == expected_stripe_protection_drives, \
        "Unexpected stripe_protection_drives value"
