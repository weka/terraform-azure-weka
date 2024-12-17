package weka_api

import (
	"github.com/weka/go-cloud-lib/lib/weka"
	"weka-deployment/common"
)

var validMethods = []weka.JrpcMethod{weka.JrpcStatus}

type WekaApiRequest struct {
	Method   weka.JrpcMethod   `json:"method"`
	Params   map[string]string `json:"params"`
	Protocol string            `json:"protocol"`

	Username          string   `json:"username"`
	Password          string   `json:"password"`
	BackendPrivateIps []string `json:"backend_private_ips"`

	vmssParams  *common.ScaleSetParams
	keyvaultURI string
}

func isSupportedMethod(method weka.JrpcMethod) bool {
	for _, m := range validMethods {
		if method == m {
			return true
		}
	}
	return false
}
