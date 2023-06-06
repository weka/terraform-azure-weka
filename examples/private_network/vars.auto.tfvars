prefix                      = "weka"
rg_name                     = "weka-rg"
address_space               = "10.0.0.0/16"
subnet_prefixes             = ["10.0.2.0/24","10.0.3.0/24","10.0.4.0/24","10.0.5.0/24"]
subnets_delegation_prefixes = ["10.0.1.0/25"]
cluster_name                = "poc"
private_network             = true
set_obs_integration         = true
tiering_ssd_percent         = 20
cluster_size                = 6
instance_type               = "Standard_L8s_v3"
apt_repo_url                = "http://11.0.0.4/ubuntu/mirror/archive.ubuntu.com/ubuntu/"
install_weka_url            = "https://wekadeploytars.blob.core.windows.net/tars/weka-4.2.0.86-beta.tar"
install_ofed_url            = "https://wekadeploytars.blob.core.windows.net/tars/MLNX_OFED_LINUX-5.8-1.1.2.1-ubuntu20.04-x86_64.tgz"