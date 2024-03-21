package terminate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
)

type VmInfo struct {
	HostName   string
	InstanceId string
}

type instancesMap map[string]*common.VMInfoSummary

func instancesToMap(instances []*common.VMInfoSummary) instancesMap {
	im := make(instancesMap)
	for _, instance := range instances {
		im[common.GetScaleSetVmId(instance.ID)] = instance
	}
	return im
}

func getDeltaInstancesIds(ctx context.Context, vmssParams *common.ScaleSetParams, scaleResponse protocol.ScaleResponse) (deltaInstanceIDs []string, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("Getting delta instances")
	netInterfaces, err := common.GetScaleSetVmsNetworkPrimaryNICs(ctx, vmssParams, nil)
	if err != nil {
		return
	}
	instanceIdPrivateIp := map[string]string{}

	for _, ni := range netInterfaces {
		id := common.GetScaleSetVmId(*ni.Properties.VirtualMachine.ID)
		privateIp := *ni.Properties.IPConfigurations[0].Properties.PrivateIPAddress
		instanceIdPrivateIp[id] = privateIp
	}

	logger.Info().Msgf("Found %d instances on scale set", len(instanceIdPrivateIp))
	instanceIpsSet := common.GetInstanceIpsSet(scaleResponse)
	logger.Debug().Msgf("Instance id to private ip map:%s", instanceIdPrivateIp)
	logger.Debug().Msgf("Scale response Instance ips set:%s", instanceIpsSet)

	for id, privateIp := range instanceIdPrivateIp {
		if _, ok := instanceIpsSet[privateIp]; !ok {
			deltaInstanceIDs = append(deltaInstanceIDs, id)
		}
	}
	logger.Info().Msgf("Delta instances%s", deltaInstanceIDs)
	return
}

func setForExplicitRemoval(instance *common.VMInfoSummary, toRemove []protocol.HgInstance) bool {
	for _, i := range toRemove {
		if common.GetScaleSetVmId(instance.ID) == i.Id {
			return true
		}
	}
	return false
}

func getInstanceCreationTime(instance *common.VMInfoSummary) (provisionTime *time.Time) {
	for _, status := range instance.InstanceViewStatuses {
		if *status.Code == "ProvisioningState/succeeded" {
			provisionTime = status.Time
			return
		}
	}
	return
}

func terminateUnneededInstances(ctx context.Context, vmssParams *common.ScaleSetParams, instances []*common.VMInfoSummary, explicitRemoval []protocol.HgInstance) (terminatedInstancesMap instancesMap, errs []error) {
	logger := logging.LoggerFromCtx(ctx)

	terminateInstanceIds := make([]string, 0)
	imap := instancesToMap(instances)

	for _, instance := range instances {
		logger.Info().Msgf("Handling instance %s(%s) removal", instance.Name, instance.InstanceID)
		if !setForExplicitRemoval(instance, explicitRemoval) {
			instanceCreationTime := getInstanceCreationTime(instance)
			if instanceCreationTime == nil {
				logger.Info().Msgf("Couldn't retrieve instance %s creation time, it is probably too new, giving grace time before removal", instance.InstanceID)
				continue
			}
			if time.Since(*instanceCreationTime) < time.Minute*30 {
				logger.Info().Msgf("Instance %s is not explicitly set for removal, giving 30M grace time", instance.InstanceID)
				continue
			}
		}
		instanceState := common.GetInstancePowerState(instance)
		if instanceState == "running" || instanceState == "starting" {
			terminateInstanceIds = append(terminateInstanceIds, instance.InstanceID)
		}
	}

	terminatedInstances, errs := common.TerminateScaleSetInstances(ctx, vmssParams, terminateInstanceIds)
	terminatedInstancesMap = make(instancesMap)
	for _, id := range terminatedInstances {
		terminatedInstancesMap[id] = imap[id]
	}
	return
}

func terminateUnhealthyInstances(ctx context.Context, vmssParams *common.ScaleSetParams, toTerminate []string) []error {
	_, terminateErrors := common.TerminateScaleSetInstances(ctx, vmssParams, toTerminate)
	return terminateErrors
}

func setDeletionProtection(ctx context.Context, allVms []*common.VMInfoSummary, excludeInstanceIds []string, vmssParams *common.ScaleSetParams, stateParams common.BlobObjParams) {
	logger := logging.LoggerFromCtx(ctx)

	// check deletion protection
	var vmsWithoutProtection []VmInfo

	for _, vm := range allVms {
		protectionPolicyExists := vm.ProtectionPolicy != nil && vm.ProtectionPolicy.ProtectFromScaleSetActions != nil
		protected := protectionPolicyExists && *vm.ProtectionPolicy.ProtectFromScaleSetActions
		instanceId := vm.InstanceID

		for _, excludedId := range excludeInstanceIds {
			if instanceId == excludedId {
				logger.Debugf("Instance %s is chosen for termination, no need to protect", instanceId)
				continue
			}
		}
		if !protected && vm.ComputerName != nil {
			vmInfo := VmInfo{HostName: *vm.ComputerName, InstanceId: instanceId}
			vmsWithoutProtection = append(vmsWithoutProtection, vmInfo)
		}
	}
	logger.Info().Msgf("%d VMs are not protected from deletion", len(vmsWithoutProtection))

	// set deletion protection in case it's not set
	for _, vm := range vmsWithoutProtection {
		logger.Info().Msgf("Setting deletion protection for VM %v", vm)
		// do not retry, but report
		common.RetrySetDeletionProtectionAndReport(ctx, vmssParams, stateParams, vm.InstanceId, vm.HostName, 0, time.Second)
	}
}

func Terminate(ctx context.Context, scaleResponse protocol.ScaleResponse, vmssParams *common.ScaleSetParams, stateParams common.BlobObjParams) (response protocol.TerminatedInstancesResponse, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger = logger.WithStrValue("vmss", vmssParams.ScaleSetName)
	logger.Info().Msg("Running termination function...")

	response.Version = protocol.Version

	if scaleResponse.Version != protocol.Version {
		err = errors.New("incompatible scale response version")
		return
	}

	if vmssParams.ScaleSetName == "" {
		err = errors.New("vmScaleSetName is empty")
		return
	}
	if len(scaleResponse.Hosts) == 0 {
		err = errors.New("hosts list must not be empty")
		return
	}

	response.TransientErrors = scaleResponse.TransientErrors[0:len(scaleResponse.TransientErrors):len(scaleResponse.TransientErrors)]

	// get VMs expanded list which will be used later
	vms, err := common.GetScaleSetVmsExpandedView(ctx, vmssParams)
	if err != nil {
		err = fmt.Errorf("cannot get VMs list for vmss %s: %v", vmssParams.ScaleSetName, err)
		return
	}

	unhealthyInstanceIds := common.GetUnhealthyInstancesToTerminate(ctx, vms)
	errs := terminateUnhealthyInstances(ctx, vmssParams, unhealthyInstanceIds)
	response.AddTransientErrors(errs)

	logger.Info().Msgf("Instances set for explicit removal: %s", scaleResponse.ToTerminate)
	deltaInstanceIds, err := getDeltaInstancesIds(ctx, vmssParams, scaleResponse)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	// NOTE: we want to have deletion protection set for all instances (which are not selected for termination)
	// in order to avoid races, this step is presented here, during "terminate" step
	unprotectedVmIds := append(deltaInstanceIds, unhealthyInstanceIds...)
	setDeletionProtection(ctx, vms, unprotectedVmIds, vmssParams, stateParams)

	if len(deltaInstanceIds) == 0 {
		logger.Info().Msgf("No delta instances ids")
		return
	}

	candidatesToTerminate, err := common.FilterSpecificScaleSetInstances(ctx, vms, deltaInstanceIds)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	terminatedInstancesMap, errs := terminateUnneededInstances(ctx, vmssParams, candidatesToTerminate, scaleResponse.ToTerminate)
	response.AddTransientErrors(errs)

	for instanceId, instance := range terminatedInstancesMap {
		terminatedInstance := protocol.TerminatedInstance{
			InstanceId: instanceId,
		}

		instanceCreationTime := getInstanceCreationTime(instance)
		if instanceCreationTime != nil {
			terminatedInstance.Creation = *instanceCreationTime
		}
		response.Instances = append(response.Instances, terminatedInstance)
	}

	return
}

func Handler(w http.ResponseWriter, r *http.Request) {
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	stateBlobName := os.Getenv("STATE_BLOB_NAME")
	nfsStateContainerName := os.Getenv("NFS_STATE_CONTAINER_NAME")
	nfsStateBlobName := os.Getenv("NFS_STATE_BLOB_NAME")
	nfsScaleSetName := os.Getenv("NFS_VMSS_NAME")

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	var invokeRequest common.InvokeRequest

	d := json.NewDecoder(r.Body)

	if err := d.Decode(&invokeRequest); err != nil {
		logger.Error().Msg("Bad request")
		common.WriteErrorResponse(w, err)
		return
	}

	var reqData map[string]interface{}

	if err := json.Unmarshal(invokeRequest.Data["req"], &reqData); err != nil {
		logger.Error().Msg("Bad request")
		common.WriteErrorResponse(w, err)
		return
	}

	var scaleResponse protocol.ScaleResponse

	if err := json.Unmarshal([]byte(reqData["Body"].(string)), &scaleResponse); err != nil {
		logger.Error().Msgf("Failed to parse scaleResponse: %s", reqData["Body"].(string))
		common.WriteErrorResponse(w, err)
		return
	}

	stateParams := common.BlobObjParams{
		StorageName:   stateStorageName,
		ContainerName: stateContainerName,
		BlobName:      stateBlobName,
	}
	vmssParams := &common.ScaleSetParams{
		SubscriptionId:    subscriptionId,
		ResourceGroupName: resourceGroupName,
		ScaleSetName:      vmScaleSetName,
		Flexible:          false,
	}
	terminateResponse, err := Terminate(ctx, scaleResponse, vmssParams, stateParams)
	if err != nil {
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	// terminate NFS instances (if NFS is configured)
	if nfsScaleSetName != "" {
		nfsParams := common.BlobObjParams{
			StorageName:   stateStorageName,
			ContainerName: nfsStateContainerName,
			BlobName:      nfsStateBlobName,
		}
		nfsVmssParams := &common.ScaleSetParams{
			SubscriptionId:    subscriptionId,
			ResourceGroupName: resourceGroupName,
			ScaleSetName:      nfsScaleSetName,
			Flexible:          true,
		}
		nfsTerminateResponse, err := Terminate(ctx, scaleResponse, nfsVmssParams, nfsParams)
		if err != nil {
			logger.Error().Err(err).Send()
			common.WriteErrorResponse(w, err)
			return
		}
		// merge responses
		terminateResponse.Instances = append(terminateResponse.Instances, nfsTerminateResponse.Instances...)
		terminateResponse.TransientErrors = append(terminateResponse.TransientErrors, nfsTerminateResponse.TransientErrors...)
	}

	common.WriteSuccessResponse(w, terminateResponse)
}
