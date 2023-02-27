package scale_down

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	strings2 "strings"
	"time"
	"weka-deployment/common"
	"weka-deployment/connectors"
	"weka-deployment/lib/jrpc"
	"weka-deployment/lib/math"
	"weka-deployment/lib/strings"
	"weka-deployment/lib/types"
	"weka-deployment/lib/weka"
	"weka-deployment/protocol"

	"github.com/google/uuid"
)

type hostState int

const unhealthyDeactivateTimeout = 120 * time.Minute
const backendCleanupDelay = 5 * time.Minute // Giving own HG chance to take care
const downKickOutTimeout = 3 * time.Hour

func (h hostState) String() string {
	switch h {
	case DEACTIVATING:
		return "DEACTIVATING"
	case HEALTHY:
		return "HEALTHY"
	case UNHEALTHY:
		return "UNHEALTHY"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", h)
	}
}

const (
	/*
		Order matters, it defines priority of hosts removal
	*/
	DEACTIVATING hostState = iota
	UNHEALTHY
	HEALTHY
)

type driveMap map[weka.DriveId]weka.Drive
type nodeMap map[weka.NodeId]weka.Node
type hostInfo struct {
	weka.Host
	id         weka.HostId
	drives     driveMap
	nodes      nodeMap
	scaleState hostState
}

func (host hostInfo) belongsToHgIpBased(instances []protocol.HgInstance) bool {
	for _, instance := range instances {
		if host.HostIp == instance.PrivateIp {
			return true
		}
	}
	return false
}

func (host hostInfo) numNotHealthyDrives() int {
	notActive := 0
	for _, drive := range host.drives {
		if strings.AnyOf(drive.Status, "INACTIVE") && time.Since(host.AddedTime) > time.Minute*5 {
			notActive += 1
		}
	}
	return notActive
}

func (host hostInfo) allDisksBeingRemoved() bool {
	ret := false
	for _, drive := range host.drives {
		ret = true
		if drive.ShouldBeActive {
			return false
		}
	}
	return ret
}

func (host hostInfo) anyDiskBeingRemoved() bool {
	for _, drive := range host.drives {
		if !drive.ShouldBeActive {
			return true
		}
	}
	return false
}

func (host hostInfo) allDrivesInactive() bool {
	for _, drive := range host.drives {
		if drive.Status != "INACTIVE" {
			return false
		}
	}
	return true
}

func (host hostInfo) managementTimedOut(timeout time.Duration) bool {
	for nodeId, node := range host.nodes {
		if !nodeId.IsManagement() {
			continue
		}
		var period time.Time
		if node.LastFencingTime != nil {
			period = *node.LastFencingTime
		} else {
			period = host.StateChangedTime
		}
		if node.Status == "DOWN" && time.Since(period) > timeout {
			return true
		}
	}
	return false
}

func remoteDownHosts(hosts []hostInfo, jpool *jrpc.Pool) {

}

func getNumToDeactivate(ctx context.Context, hostInfo []hostInfo, desired int) int {
	/*
		A - Fully active, healthy
		T - Target state
		U - Unhealthy, we want to remove it for whatever reason. DOWN host, FAILED drive, so on
		D - Drives/hosts being deactivated
		new_D - Decision to start deactivating, i.e transition to D, basing on U. Never more then 2 for U

		new_D = func(A, U, T, D)

		new_D = max(A+U+D-T, min(2-D, U), 0)
	*/
	logger := common.LoggerFromCtx(ctx)

	nHealthy := 0
	nUnhealthy := 0
	nDeactivating := 0

	for _, host := range hostInfo {
		switch host.scaleState {
		case HEALTHY:
			nHealthy++
		case UNHEALTHY:
			nUnhealthy++
		case DEACTIVATING:
			nDeactivating++
		}
	}

	toDeactivate := CalculateDeactivateTarget(nHealthy, nUnhealthy, nDeactivating, desired)
	logger.Info().Msgf("%d hosts set to deactivate. nHealthy: %d nUnhealthy:%d nDeactivating: %d desired:%d", toDeactivate, nHealthy, nUnhealthy, nDeactivating, desired)
	return toDeactivate
}

func getContainersDeactivationOrder(hostInfo []hostInfo) (deactivationOrderedContainer []hostInfo) {
	hostsIpsSet := make(map[string]types.Nilt)
	for _, host := range hostInfo {
		if _, ok := hostsIpsSet[host.HostIp]; !ok {
			hostsIpsSet[host.HostIp] = types.Nilv
			deactivationOrderedContainer = append(deactivationOrderedContainer, host)
		}
	}
	return
}

func CalculateDeactivateTarget(nHealthy int, nUnhealthy int, nDeactivating int, desired int) int {
	ret := math.Max(nHealthy+nUnhealthy+nDeactivating-desired, math.Min(2-nDeactivating, nUnhealthy))
	ret = math.Max(nDeactivating, ret)
	return ret
}

func isAllowedToScale(status weka.StatusResponse) error {
	if status.IoStatus != "STARTED" {
		return errors.New(fmt.Sprintf("io status:%s, aborting scale", status.IoStatus))
	}

	if status.Upgrade != "" {
		return errors.New("upgrade is running, aborting scale")
	}
	return nil
}

func deriveHostState(ctx context.Context, host *hostInfo) hostState {
	logger := common.LoggerFromCtx(ctx)

	if strings2.Contains(host.ContainerName, "drive") && host.allDisksBeingRemoved() {
		logger.Info().Msgf("Marking %s as deactivating due to unhealthy disks", host.id.String())
		return DEACTIVATING
	}
	if strings.AnyOf(host.State, "DEACTIVATING", "REMOVING", "INACTIVE") {
		return DEACTIVATING
	}
	if strings.AnyOf(host.Status, "DOWN", "DEGRADED") && host.managementTimedOut(unhealthyDeactivateTimeout) {
		logger.Info().Msgf("Marking %s as unhealthy due to DOWN", host.id.String())
		return UNHEALTHY
	}
	if host.numNotHealthyDrives() > 0 || host.anyDiskBeingRemoved() {
		logger.Info().Msgf("Marking %s as unhealthy due to unhealthy drives", host.id.String())
		return UNHEALTHY
	}
	return HEALTHY
}

func calculateHostsState(ctx context.Context, hosts []hostInfo) {
	for i := range hosts {
		host := &hosts[i]
		host.scaleState = deriveHostState(ctx, host)
	}
}

func selectInstanceByIp(ip string, instances []protocol.HgInstance) *protocol.HgInstance {
	for _, i := range instances {
		if i.PrivateIp == ip {
			return &i
		}
	}
	return nil
}

func removeContainer(ctx context.Context, jpool *jrpc.Pool, hostId int, p *protocol.ScaleResponse) (err error) {
	logger := common.LoggerFromCtx(ctx)
	err = jpool.Call(weka.JrpcRemoveHost, types.JsonDict{
		"host_id": hostId,
		"no_wait": true,
	}, nil)
	if err != nil {
		logger.Error().Err(err)
		p.AddTransientError(err, "removeInactive")
	}
	return
}

func removeInactive(ctx context.Context, hostsApiList weka.HostListResponse, hosts []hostInfo, jpool *jrpc.Pool, instances []protocol.HgInstance, p *protocol.ScaleResponse) {
	for _, host := range hosts {
		jpool.Drop(host.HostIp)
		containers := getMachineContainers(hostsApiList, host)
		err1 := removeContainer(ctx, jpool, containers.Drive.Int(), p)
		err2 := removeContainer(ctx, jpool, containers.Compute.Int(), p)
		err3 := removeContainer(ctx, jpool, containers.Frontend.Int(), p)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}

		instance := selectInstanceByIp(host.HostIp, instances)
		if instance != nil {
			p.ToTerminate = append(p.ToTerminate, *instance)
		}

		for _, drive := range host.drives {
			removeDrive(ctx, jpool, drive, p)
		}
	}
	return
}

func removeOldDrives(ctx context.Context, drives weka.DriveListResponse, jpool *jrpc.Pool, p *protocol.ScaleResponse) {
	for _, drive := range drives {
		if drive.HostId.Int() == -1 && drive.Status == "INACTIVE" {
			removeDrive(ctx, jpool, drive, p)
		}
	}
}

func removeDrive(ctx context.Context, jpool *jrpc.Pool, drive weka.Drive, p *protocol.ScaleResponse) {
	logger := common.LoggerFromCtx(ctx)

	err := jpool.Call(weka.JrpcRemoveDrive, types.JsonDict{
		"drive_uuids": []uuid.UUID{drive.Uuid},
	}, nil)
	if err != nil {
		logger.Error().Err(err)
		p.AddTransientError(err, "removeDrive")
	}
}

type machineContainers struct {
	Compute  weka.HostId `json:"compute"`
	Frontend weka.HostId `json:"frontend"`
	Drive    weka.HostId `json:"drive"`
}

func getMachineContainers(hostsApiList weka.HostListResponse, host hostInfo) (containers machineContainers) {
	for hostId, _ := range hostsApiList {
		if hostId.Int() == host.id.Int() {
			if strings2.Contains(host.ContainerName, "compute") {
				containers.Compute = hostId
			} else if strings2.Contains(host.ContainerName, "frontend") {
				containers.Frontend = hostId
			} else {
				containers.Drive = hostId
			}
		}
	}
	return
}

func ScaleDown(ctx context.Context, info protocol.HostGroupInfoResponse) (response protocol.ScaleResponse, err error) {
	/*
		Code in here based on following logic:

		A - Fully active, healthy
		T - Desired target number
		U - Unhealthy, we want to remove it for whatever reason. DOWN host, FAILED drive, so on
		D - Drives/hosts being deactivated
		NEW_D - Decision to start deactivating, i.e transition to D, basing on U. Never more then 2 for U

		NEW_D = func(A, U, T, D)

		NEW_D = max(A+U+D-T, min(2-D, U), 0)
	*/
	logger := common.LoggerFromCtx(ctx)

	response.Version = protocol.Version

	jrpcBuilder := func(ip string) *jrpc.BaseClient {
		return connectors.NewJrpcClient(ctx, ip, weka.ManagementJrpcPort, info.Username, info.Password)
	}
	ips := info.BackendIps
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(ips), func(i, j int) { ips[i], ips[j] = ips[j], ips[i] })
	jpool := &jrpc.Pool{
		Ips:     ips,
		Clients: map[string]*jrpc.BaseClient{},
		Active:  "",
		Builder: jrpcBuilder,
		Ctx:     ctx,
	}

	systemStatus := weka.StatusResponse{}
	machinesApiList := weka.MachineListResponse{}
	hostsApiList := weka.HostListResponse{}
	driveApiList := weka.DriveListResponse{}
	nodeApiList := weka.NodeListResponse{}

	err = jpool.Call(weka.JrpcStatus, struct{}{}, &systemStatus)
	if err != nil {
		return
	}
	err = isAllowedToScale(systemStatus)
	if err != nil {
		return
	}
	err = jpool.Call(weka.JrpcHostList, struct{}{}, &hostsApiList)
	if err != nil {
		return
	}
	if info.Role == "backend" {
		err = jpool.Call(weka.JrpcDrivesList, struct{}{}, &driveApiList)
		if err != nil {
			return
		}
	}
	err = jpool.Call(weka.JrpcNodeList, struct{}{}, &nodeApiList)
	if err != nil {
		return
	}

	hosts := map[weka.HostId]hostInfo{}
	for hostId, host := range hostsApiList {
		hosts[hostId] = hostInfo{
			Host:   host,
			id:     hostId,
			drives: driveMap{},
			nodes:  nodeMap{},
		}
	}
	for driveId, drive := range driveApiList {
		if _, ok := hosts[drive.HostId]; ok {
			hosts[drive.HostId].drives[driveId] = drive
		}
	}

	for nodeId, node := range nodeApiList {
		if _, ok := hosts[node.HostId]; ok {
			hosts[node.HostId].nodes[nodeId] = node
		}
	}

	var hostsList []hostInfo
	var inactiveHosts []hostInfo
	var downHosts []hostInfo

	inactiveOrDownHostsIps := make(map[string]types.Nilt)
	for _, host := range hosts {
		if _, ok := inactiveOrDownHostsIps[host.HostIp]; ok {
			continue
		}

		switch host.State {
		case "INACTIVE":
			if host.belongsToHgIpBased(info.Instances) {
				inactiveHosts = append(inactiveHosts, host)
				inactiveOrDownHostsIps[host.HostIp] = types.Nilv
				continue
			} else {
				if info.Role == "backend" {
					logger.Info().Msgf("host %s is inactive and does not belong to HG, removing from cluster", host.id)
					inactiveHosts = append(inactiveHosts, host)
					inactiveOrDownHostsIps[host.HostIp] = types.Nilv
					continue
				}
			}
		default:
			if host.belongsToHgIpBased(info.Instances) {
				hostsList = append(hostsList, host)
				continue
			}
		}

		switch host.Status {
		case "DOWN":
			logger.Info().Msgf("found down host %s %s %s", host.id, host.Aws.InstanceId, host.HostIp)
			if info.Role == "backend" {
				if host.State != "INACTIVE" && host.managementTimedOut(downKickOutTimeout) {
					logger.Info().Msgf("host %s is still active but down for too long, kicking out", host.id)
					downHosts = append(downHosts, host)
					inactiveOrDownHostsIps[host.HostIp] = types.Nilv
					continue
				}
			}
		}

	}

	calculateHostsState(ctx, hostsList)

	sort.Slice(hostsList, func(i, j int) bool {
		// Giving priority to disks to hosts with disk being removed
		// Then hosts with disks not in active state
		// Then hosts sorted by add time
		a := hostsList[i]
		b := hostsList[j]
		if a.scaleState < b.scaleState {
			return true
		}
		if a.scaleState > b.scaleState {
			return false
		}
		if a.numNotHealthyDrives() > b.numNotHealthyDrives() {
			return true
		}
		if a.numNotHealthyDrives() < b.numNotHealthyDrives() {
			return false
		}
		return a.AddedTime.Before(b.AddedTime)
	})

	removeInactive(ctx, hostsApiList, inactiveHosts, jpool, info.Instances, &response)
	removeOldDrives(ctx, driveApiList, jpool, &response)

	err = jpool.Call(weka.JrpcMachinesList, struct{}{}, &machinesApiList)
	if err != nil {
		return
	}
	deactivationOrderedContainer := getContainersDeactivationOrder(hostsList)
	numToDeactivate := len(deactivationOrderedContainer) - info.DesiredCapacity

	deactivateHost := func(host hostInfo) {
		logger.Info().Msgf("Trying to deactivate host %s", host.id)
		for _, drive := range host.drives {
			if drive.ShouldBeActive {
				err := jpool.Call(weka.JrpcDeactivateDrives, types.JsonDict{
					"drive_uuids": []uuid.UUID{drive.Uuid},
				}, nil)
				if err != nil {
					logger.Error().Err(err)
					response.AddTransientError(err, "deactivateDrive")
				}
			}
		}

		containers := getMachineContainers(hostsApiList, host)
		err := jpool.Call(weka.JrpcDeactivateHosts, types.JsonDict{
			"host_ids": []weka.HostId{
				containers.Drive,
				containers.Compute,
				containers.Frontend,
			},
			"skip_resource_validation": false,
		}, nil)
		if err != nil {
			logger.Error().Err(err)
			response.AddTransientError(err, "deactivateHost")
		} else {
			jpool.Drop(host.HostIp)
		}

	}

	for _, host := range deactivationOrderedContainer[:numToDeactivate] {
		deactivateHost(host)
	}

	for _, host := range downHosts {
		deactivateHost(host)
	}

	for _, host := range hostsList {
		response.Hosts = append(response.Hosts, protocol.ScaleResponseHost{
			InstanceId: host.Aws.InstanceId,
			PrivateIp:  host.HostIp,
			State:      host.State,
			AddedTime:  host.AddedTime,
			HostId:     host.id,
		})
	}

	return
}

func Handler(w http.ResponseWriter, r *http.Request) {
	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})
	var invokeRequest common.InvokeRequest

	ctx := r.Context()
	logger := common.LoggerFromCtx(ctx)

	d := json.NewDecoder(r.Body)
	err := d.Decode(&invokeRequest)
	if err != nil {
		logger.Error().Msg("Bad request")
		return
	}

	var reqData map[string]interface{}
	err = json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		logger.Error().Msg("Bad request")
		return
	}

	var info protocol.HostGroupInfoResponse

	if json.Unmarshal([]byte(reqData["Body"].(string)), &info) != nil {
		logger.Error().Msg("Bad request")
		return
	}

	scaleResponse, err := ScaleDown(ctx, info)
	if err != nil {
		resData["body"] = err.Error()
	} else {
		resData["body"] = scaleResponse
	}
	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
