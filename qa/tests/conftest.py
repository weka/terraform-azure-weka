import pytest


def pytest_addoption(parser):
    parser.addoption('--get_weka_io_token', action='store', default=None, help='Weka io token')
    parser.addoption('--client_id', action='store', default=None,
                     help='Client ID for autotest Azure SP.')
    parser.addoption('--client_secret', action='store', default=None,
                     help='Client secret (password) for autotest Azure SP.')
    parser.addoption('--tenant_id', action='store', default=None,
                     help='Tenant ID.')
    parser.addoption('--subscription_id', action='store', default=None, help='Subscription ID from Azure cloud')
    parser.addoption('--cloud', action='store', default='Azure', help='Specify cloud provider name (default=Azure)')


@pytest.fixture(scope="session", autouse=True)
def command_line_args(request):
    kwargs = {'client_id': request.config.getoption('--client_id'),
              'client_secret': request.config.getoption('--client_secret'),
              'tenant_id': request.config.getoption('--tenant_id'),
              'get_weka_io_token': request.config.getoption('--get_weka_io_token'),
              'subscription_id': request.config.getoption('--subscription_id'),
              'cloud': request.config.getoption('--cloud')
              }
    return kwargs
