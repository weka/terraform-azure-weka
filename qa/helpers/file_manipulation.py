from qa.helpers.core import ShellClient, BASE_DIR


def extend_tf_variables(example_dir_name, additional_variable_file_name):
    cmd = f'cat {BASE_DIR}/qa/test_data/tf_variables/{additional_variable_file_name} >> ' \
          f'{BASE_DIR}/examples/{example_dir_name}/variables.tf'
    shell = ShellClient()
    shell.execute_command(cmd)
