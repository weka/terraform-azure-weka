import requests
from requests.exceptions import ReadTimeout

def get_cluster_status(prefix, cluster_name, function_key):
    try:
        return requests.get(url=f'https://{prefix}-{cluster_name}-function-app.azurewebsites.net/api/status',
                        params={"code": function_key})
    except ReadTimeout:
        return requests.get(url=f'https://{prefix}-{cluster_name}-function-app.azurewebsites.net/api/status',
                            params={"code": function_key})