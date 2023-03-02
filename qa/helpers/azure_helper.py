from azure.identity import ClientSecretCredential
from azure.mgmt.resource import ResourceManagementClient
from azure.mgmt.web import WebSiteManagementClient
from qa.helpers.core import logger


class AzureHelper:
    def __init__(self, **kwargs):
        self.__credentials = ClientSecretCredential(kwargs.get('tenant_id'),
                                                    kwargs.get('client_id'),
                                                    kwargs.get('client_secret'))
        self.__subscription_id = kwargs.get('subscription_id')
        self.__resource_client = ResourceManagementClient(self.__credentials, self.__subscription_id)

    def create_resource_group(self, name, location="eastus"):
        try:
            self.__resource_client.resource_groups.create_or_update(name, {"location": location})
        except Exception as e:
            logger.error(f'During creation resource group an exception occurs: {e}')
            raise e

    def delete_resource_group(self, name):
        try:
            self.__resource_client.resource_groups.begin_delete(name)
        except Exception as e:
            logger.error(f'During deletion resource group an exception occurs: {e}')
            raise e

    def get_function_key(self, rg, prefix, cluster_name):
        web_client = WebSiteManagementClient(subscription_id=self.__subscription_id, credential=self.__credentials)
        response = web_client.web_apps.list_function_keys(rg, f'{prefix}-{cluster_name}-function-app', 'status')
        return response.additional_properties.get('default')
