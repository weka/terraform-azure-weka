package weka_api

import (
	"context"
	"encoding/json"
	"github.com/weka/go-cloud-lib/connectors"
	"github.com/weka/go-cloud-lib/lib/jrpc"
	"github.com/weka/go-cloud-lib/lib/weka"
	"github.com/weka/go-cloud-lib/logging"
	"math/rand"
	"time"
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

	vmssParams *common.ScaleSetParams
}

func isSupportedMethod(method weka.JrpcMethod) bool {
	for _, m := range validMethods {
		if method == m {
			return true
		}
	}
	return false
}

func CallJRPC(ctx context.Context, request WekaApiRequest) (json.RawMessage, error) {
	logger := logging.LoggerFromCtx(ctx)

	jrpcBuilder := func(ip string) *jrpc.BaseClient {
		return connectors.NewJrpcClient(ctx, ip, weka.ManagementJrpcPort, request.Username, request.Password)
	}

	vmIps, err := common.GetVmsPrivateIps(ctx, request.vmssParams)
	if err != nil {
		return nil, err
	}
	ips := make([]string, 0, len(vmIps))
	for _, ip := range vmIps {
		ips = append(ips, ip)
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(ips), func(i, j int) { ips[i], ips[j] = ips[j], ips[i] })
	logger.Info().Msgf("ips: %s", ips)
	jpool := &jrpc.Pool{
		Ips:     ips,
		Clients: map[string]*jrpc.BaseClient{},
		Active:  "",
		Builder: jrpcBuilder,
		Ctx:     ctx,
	}

	var rawWekaStatus json.RawMessage

	err = jpool.Call(request.Method, struct{}{}, &rawWekaStatus)
	if err != nil {
		return nil, err
	}
	return rawWekaStatus, nil

}
