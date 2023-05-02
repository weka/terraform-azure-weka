package bash_functions

import (
	"github.com/lithammer/dedent"
)

func GetCoreIds() string {
	s := `
	core_ids=$(cat /sys/devices/system/cpu/cpu*/topology/thread_siblings_list | cut -d "-" -f 1 | sort -u | tr '\n' ' ')
	core_ids="${core_ids[@]/0}"
	IFS=', ' read -r -a core_ids <<< "$core_ids"
	core_idx_begin=0
	get_core_ids() {
		core_idx_end=$(($core_idx_begin + $1))
		res=${core_ids[i]}
		for (( i=$(($core_idx_begin + 1)); i<$core_idx_end; i++ ))
		do
			res=$res,${core_ids[i]}
		done
		core_idx_begin=$core_idx_end
		eval "$2=$res"
	}
	`
	return dedent.Dedent(s)
}

func GetNetStrForDpdk() string {
	s := `
	function getNetStrForDpdk() {
		i=$1
		j=$2
		net=" "
		gateway=$(route -n | grep 0.0.0.0 | grep UG | awk '{print $2}')
		for ((i; i<$j; i++)); do
			eth=$(ifconfig | grep eth$i -C2 | grep 'inet ' | awk '{print $2}')
			if [ $eth eq "" ];then
				net=""
				break
			fi
			enp=$(ls -l /sys/class/net/eth$i/ | grep lower | awk -F"_" '{print $2}' | awk '{print $1}')
			bits=$(ip -o -f inet addr show eth$i | awk '{print $4}')
			IFS='/' read -ra netmask <<< "$bits"
			net="$net --net $enp/$eth/${netmask[1]}/$gateway"
		done
	}
	`
	return dedent.Dedent(s)
}

func GetHashedPrivateIpBashCmd() string {
	return "printf $(hostname -I) | sha256sum | tr -d '-' | cut -c1-16"
}
