package weka_api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"math/rand"
	"net/http"
	"os"
	"time"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/connectors"
	"github.com/weka/go-cloud-lib/lib/jrpc"
	"github.com/weka/go-cloud-lib/lib/weka"
	"github.com/weka/go-cloud-lib/logging"
)

func (wr *WekaApiRequest) MakeRequest(ctx context.Context) (*json.RawMessage, error) {
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	nfsScaleSetName := os.Getenv("NFS_VMSS_NAME")

	wr.keyvaultURI = os.Getenv("KEY_VAULT_URI")

	wr.vmssParams = &common.ScaleSetParams{
		SubscriptionId:    subscriptionId,
		ResourceGroupName: resourceGroupName,
		ScaleSetName:      common.GetVmScaleSetName(prefix, clusterName),
		Flexible:          false,
	}

	if wr.Protocol == "nfs" {
		wr.vmssParams.ScaleSetName = nfsScaleSetName
		wr.vmssParams.Flexible = true
	}

	return CallJRPC(ctx, wr)
}

func CallJRPC(ctx context.Context, wekaApi *WekaApiRequest) (message *json.RawMessage, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("fetching cluster status...")

	credentials, err := common.GetWekaClusterCredentials(ctx, wekaApi.keyvaultURI)
	if err != nil {
		return
	}

	jrpcBuilder := func(ip string) *jrpc.BaseClient {
		return connectors.NewJrpcClient(ctx, ip, weka.ManagementJrpcPort, credentials.Username, credentials.Password)
	}

	vmIps, err := common.GetVmsPrivateIps(ctx, wekaApi.vmssParams)
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

	err = jpool.Call(wekaApi.Method, struct{}{}, &rawWekaStatus)
	if err != nil {
		return nil, err
	}

	return &rawWekaStatus, nil
}

func Handler(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	//logger := logging.LoggerFromCtx(ctx)
	log.Info().Msg("this is weka api")

	var err error
	var wekaApi WekaApiRequest
	err = common.GetBody(ctx, w, r, &wekaApi)
	if !isSupportedMethod(wekaApi.Method) {
		common.WriteErrorResponse(w, fmt.Errorf("bad method"))
		return
	}

	result, err := wekaApi.MakeRequest(ctx)

	//var result *json.RawMessage
	//if wekaApi.Method == "" || wekaApi.Method == "status" {
	//	result, err = CallJRPC(ctx, wekaApi)
	//}

	if err != nil {
		common.WriteErrorResponse(w, err)
		return
	}
	common.WriteSuccessResponse(w, result)
}
