from qa.helpers.core import SSHClient


def get_weka_status_on_instance(ip_address, key_file_path):
    s = SSHClient(host=ip_address, username='weka', key_file=key_file_path)
    s.connect()
    s.exec_command('weka status')
    return s.stdout
