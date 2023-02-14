import logging
import paramiko
from pathlib import Path
from subprocess import Popen, PIPE, CalledProcessError

BASE_DIR = Path(__file__).parent.parent.parent

logger = logging.getLogger('qa-test')
logger.addHandler(logging.StreamHandler())
logger.setLevel(logging.DEBUG)


class ShellClient:
    def __init__(self):
        self.stderr = []
        self.stdout = []

    def execute_command(self, cmd):
        with Popen(cmd, stdout=PIPE, stderr=PIPE, bufsize=1, universal_newlines=True, shell=True) as p:
            for line in p.stdout:
                logger.info(line.strip())
                self.stdout.append(line.strip())
            for line in p.stderr:
                logger.error(line.strip())
                self.stderr.append(line.strip())
        if p.returncode != 0:
            raise CalledProcessError(p.returncode, p.args)


class SSHClient:

    def __init__(self, host, username, key_file):
        self.__host = host
        self.__username = username
        self.__key = paramiko.RSAKey.from_private_key_file(key_file)
        self.__client = None
        self.stdout = []
        self.stderr = []

    def connect(self):
        self.__client = paramiko.SSHClient()
        self.__client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        self.__client.connect(hostname=self.__host, username=self.__username, pkey=self.__key)

    def exec_command(self, cmd):
        stdin, stdout, stderr = self.__client.exec_command(cmd)
        exit_code = stdout.channel.recv_exit_status()
        for line in stdout:
            logger.info(line.strip())
            self.stdout.append(line.strip())
        for line in stderr:
            logger.info(line.strip())
            self.stderr.append(line.strip())

    def __del__(self):
        self.__client.close()
