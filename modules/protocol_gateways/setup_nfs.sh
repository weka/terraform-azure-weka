# create interface group
weka nfs interface-group add ${interface_group_name} NFS
# get container id
container_id=$(weka cluster container | grep frontend0 | grep ${gateways_name} | grep UP | awk '{print $1}')
# get device to use
port=$(ip -o -f inet addr show | grep -m 1 eth | awk '{print $2}')

weka nfs interface-group port add ${interface_group_name} $container_id $port
# show interface group
weka nfs interface-group

weka nfs client-group add ${client_group_name}
weka nfs rules add dns ${client_group_name} *
weka nfs permission add default ${client_group_name}
weka nfs client-group

weka local ps
echo "$(date -u): nfs setup complete"
