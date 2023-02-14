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
	InitialSize            int        `json:"initial_size"`
	DesiredSize            int        `json:"desired_size"`
	Clusterized            bool       `json:"clusterized"`
	ReadyForClusterization []string   `json:"ready_for_clusterization"`
	WekaStatus             WekaStatus `json:"weka_status"`
}

type ClusterCloud struct {
	Enabled bool   `json:"enabled"`
	Healthy bool   `json:"healthy"`
	Proxy   string `json:"proxy"`
	Url     string `json:"url"`
}

type ClusterCapacity struct {
	TotalBytes         float32 `json:"total_bytes"`
	HotSpareBytes      float32 `json:"hot_spare_bytes"`
	UnprovisionedBytes float32 `json:"unprovisioned_bytes"`
}

type ClusterNodes struct {
	BlackListed int `json:"black_listed"`
	Total       int `json:"total"`
}

type ClusterUsage struct {
	DriveCapacityGb  int `json:"drive_capacity_gb"`
	UsableCapacityGb int `json:"usable_capacity_gb"`
	ObsCapacityGb    int `json:"obs_capacity_gb"`
}

type ClusterLicensing struct {
	IoStartEligibility bool         `json:"io_start_eligibility"`
	Usage              ClusterUsage `json:"usage"`
	Mode               string       `json:"mode"`
}

type WekaStatus struct {
	HotSpare               int               `json:"hot_spare"`
	IoStatus               string            `json:"io_status"`
	Drives                 weka.HostsCount   `json:"drives"`
	Name                   string            `json:"name"`
	IoStatusChangedTime    time.Time         `json:"io_status_changed_time"`
	IoNodes                weka.HostsCount   `json:"io_nodes"`
	Cloud                  ClusterCloud      `json:"cloud"`
	ReleaseHash            string            `json:"release_hash"`
	Hosts                  weka.ClusterCount `json:"hosts"`
	StripeDataDrives       int               `json:"stripe_data_drives"`
	Release                string            `json:"release"`
	ActiveAlertsCount      int               `json:"active_alerts_count"`
	Capacity               ClusterCapacity   `json:"capacity"`
	IsCluster              bool              `json:"is_cluster"`
	Status                 string            `json:"status"`
	StripeProtectionDrives int               `json:"stripe_protection_drives"`
	Guid                   string            `json:"guid"`
	Nodes                  ClusterNodes      `json:"nodes"`
	Licensing              ClusterLicensing  `json:"licensing"`
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

	wekaStatus := WekaStatus{}
	if err = json.Unmarshal(rawWekaStatus, &wekaStatus); err != nil {
		return
	}
	clusterStatus.WekaStatus = wekaStatus

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
