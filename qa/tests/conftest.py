import pytest


def pytest_addoption(parser):
    parser.addoption('--rg_name', action='store', default=None, help='Name of Azure recourse group')
    parser.addoption('--prefix', action='store', default='qa', help='Prefix')
    parser.addoption('--cluster_name', action='store', default='wekatest', help='Name of the cluster')
    parser.addoption('--get_weka_io_token', action='store', default=None, help='Weka io token')
    parser.addoption('--client_id', action='store', default=None,
                     help='Client ID for autotest Azure SP. Provide this value only for CI testrun!')
    parser.addoption('--client_secret', action='store', default=None,
                     help='Client secret (password) for autotest Azure SP. Provide this value only for CI testrun!')
    parser.addoption('--tenant_id', action='store', default=None,
                     help='Tenant ID. Provide this value only for CI testrun!')
    parser.addoption('--subscription_id', action='store', default=None, help='Subscription ID for Azure cloud')


@pytest.fixture
def command_line_args(request):
    kwargs = {'rg_name': request.config.getoption('--rg_name'),
              'prefix': request.config.getoption('--prefix'),
              'cluster_name': request.config.getoption('--cluster_name'),
              'get_weka_io_token': request.config.getoption('--get_weka_io_token'),
              'subscription_id': request.config.getoption('--subscription_id')
              }
    # Add required attributes for Testrun in CI. Can be ignored in local testrun
    for attr in ['--client_id', '--client_secret', '--tenant_id']:
        value = request.config.getoption(attr)
        if value:
            kwargs.update({attr[2:]: value})
    return kwargs
