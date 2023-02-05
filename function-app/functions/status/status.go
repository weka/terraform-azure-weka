package status

import (
	"context"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"math/rand"
	"net/http"
	"os"
	"time"
	"weka-deployment/common"
	"weka-deployment/connectors"
	"weka-deployment/lib/jrpc"
	"weka-deployment/lib/weka"
)

type ClusterStatus struct {
	InitialSize            int                 `json:"initial_size"`
	DesiredSize            int                 `json:"desired_size"`
	Clusterized            bool                `json:"clusterized"`
	ReadyForClusterization []string            `json:"ready_for_clusterization"`
	SystemStatus           weka.StatusResponse `json:"system_status"`
	RawWekaStatus          json.RawMessage     `json:"raw_weka_status"`
}

func GetClusterStatus(subscriptionId, resourceGroupName, vmScaleSetName, stateStorageName, stateContainerName, keyVaultUri string) (clusterStatus ClusterStatus, err error) {
	log.Info().Msg("fetching cluster status...")
	state, err := common.ReadState(stateStorageName, stateContainerName)
	if err != nil {
		return
	}
	clusterStatus.InitialSize = state.InitialSize
	clusterStatus.DesiredSize = state.DesiredSize
	clusterStatus.Clusterized = state.Clusterized
	clusterStatus.ReadyForClusterization = state.Instances
	if !state.Clusterized {
		return
	}

	wekaPassword, err := common.GetWekaClusterPassword(keyVaultUri)
	if err != nil {
		return
	}

	ctx := context.Background()
	jrpcBuilder := func(ip string) *jrpc.BaseClient {
		return connectors.NewJrpcClient(ctx, ip, weka.ManagementJrpcPort, "admin", wekaPassword)
	}

	vmIps, err := common.GetVmsPrivateIps(subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		return
	}
	ips := make([]string, len(vmIps))
	for _, ip := range vmIps {
		ips = append(ips, ip)
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(ips), func(i, j int) { ips[i], ips[j] = ips[j], ips[i] })
	log.Info().Msgf("ips: %s", ips)
	jpool := &jrpc.Pool{
		Ips:     ips,
		Clients: map[string]*jrpc.BaseClient{},
		Active:  "",
		Builder: jrpcBuilder,
		Ctx:     ctx,
	}

	var rawWekaStatus json.RawMessage

	err = jpool.Call(weka.JrpcStatus, struct{}{}, &rawWekaStatus)
	if err != nil {
		return
	}
	clusterStatus.RawWekaStatus = rawWekaStatus

	systemStatus := weka.StatusResponse{}
	if err = json.Unmarshal(rawWekaStatus, &systemStatus); err != nil {
		return
	}
	clusterStatus.SystemStatus = systemStatus

	return
}

func Handler(w http.ResponseWriter, r *http.Request) {
	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})

	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)
	clusterStatus, err := GetClusterStatus(subscriptionId, resourceGroupName, vmScaleSetName, stateStorageName, stateContainerName, keyVaultUri)

	if err != nil {
		resData["body"] = err.Error()
	} else {

		resData["body"] = clusterStatus
	}
	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
