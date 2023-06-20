package terminate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"weka-deployment/common"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
)

type VmInfo struct {
	HostName   string
	InstanceId string
}

type instancesMap map[string]*armcompute.VirtualMachineScaleSetVM

func instancesToMap(instances []*armcompute.VirtualMachineScaleSetVM) instancesMap {
	im := make(instancesMap)
	for _, instance := range instances {
		im[common.GetScaleSetVmId(*instance.ID)] = instance
	}
	return im
}

func getDeltaInstancesIds(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string, scaleResponse protocol.ScaleResponse) (deltaInstanceIDs []string, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("Getting delta instances")
	netInterfaces, err := common.GetScaleSetVmsNetworkPrimaryNICs(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
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

func setForExplicitRemoval(instance *armcompute.VirtualMachineScaleSetVM, toRemove []protocol.HgInstance) bool {
	for _, i := range toRemove {
		if common.GetScaleSetVmId(*instance.ID) == i.Id {
			return true
		}
	}
	return false
}

func getInstanceCreationTime(instance *armcompute.VirtualMachineScaleSetVM) (provisionTime *time.Time) {
	for _, status := range instance.Properties.InstanceView.Statuses {
		if *status.Code == "ProvisioningState/succeeded" {
			provisionTime = status.Time
			return
		}
	}
	return
}

func getInstancePowerState(instance *armcompute.VirtualMachineScaleSetVM) (powerState string) {
	prefix := "PowerState/"
	for _, status := range instance.Properties.InstanceView.Statuses {
		if strings.HasPrefix(*status.Code, prefix) {
			powerState = strings.TrimPrefix(*status.Code, prefix)
			return
		}
	}
	return
}

func terminateUnneededInstances(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string, instances []*armcompute.VirtualMachineScaleSetVM, explicitRemoval []protocol.HgInstance) (terminatedInstancesMap instancesMap, errs []error) {
	logger := logging.LoggerFromCtx(ctx)

	terminateInstanceIds := make([]string, 0)
	imap := instancesToMap(instances)

	for _, instance := range instances {
		logger.Info().Msgf("Handling instance %s(%s) removal", *instance.Name, *instance.InstanceID)
		if !setForExplicitRemoval(instance, explicitRemoval) {
			instanceCreationTime := getInstanceCreationTime(instance)
			if instanceCreationTime == nil {
				logger.Info().Msgf("Couldn't retrieve instance %s creation time, it is probably too new, giving grace time before removal", *instance.InstanceID)
				continue
			}
			if time.Since(*instanceCreationTime) < time.Minute*30 {
				logger.Info().Msgf("Instance %s is not explicitly set for removal, giving 30M grace time", *instance.InstanceID)
				continue
			}
		}
		instanceState := getInstancePowerState(instance)
		if instanceState == "running" || instanceState == "starting" {
			terminateInstanceIds = append(terminateInstanceIds, *instance.InstanceID)
		}
	}

	terminatedInstances, errs := common.TerminateScaleSetInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, terminateInstanceIds)
	terminatedInstancesMap = make(instancesMap)
	for _, id := range terminatedInstances {
		terminatedInstancesMap[id] = imap[id]
	}
	return
}

func getUnhealthyInstancesToTerminate(ctx context.Context, scaleSetVms []*armcompute.VirtualMachineScaleSetVM) (toTerminate []string) {
	logger := logging.LoggerFromCtx(ctx)

	for _, vm := range scaleSetVms {
		if vm.Properties.InstanceView == nil || vm.Properties.InstanceView.VMHealth == nil {
			continue
		}
		healthStatus := *vm.Properties.InstanceView.VMHealth.Status.Code
		if healthStatus == "HealthState/unhealthy" {
			instanceState := getInstancePowerState(vm)
			logger.Debug().Msgf("instance state: %s", instanceState)
			if instanceState == "stopped" {
				toTerminate = append(toTerminate, common.GetScaleSetVmId(*vm.ID))
			}

		}
	}

	logger.Info().Msgf("found %d unhealthy stopped instances to terminate: %s", len(toTerminate), toTerminate)
	return
}

func terminateUnhealthyInstances(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string, toTerminate []string) []error {
	_, terminateErrors := common.TerminateScaleSetInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, toTerminate)
	return terminateErrors
}

func getScaleSetVmsExpandedView(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string) ([]*armcompute.VirtualMachineScaleSetVM, error) {
	expand := "instanceView"
	return common.GetScaleSetInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, &expand)
}

func setDeletionProtection(ctx context.Context, allVms []*armcompute.VirtualMachineScaleSetVM, excludeInstanceIds []string, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName string) {
	logger := logging.LoggerFromCtx(ctx)

	// check deletion protection
	var vmsWithoutProtection []VmInfo

	for _, vm := range allVms {
		protectionPolicyExists := vm.Properties.ProtectionPolicy != nil && vm.Properties.ProtectionPolicy.ProtectFromScaleSetActions != nil
		protected := protectionPolicyExists && *vm.Properties.ProtectionPolicy.ProtectFromScaleSetActions
		instanceId := common.GetScaleSetVmId(*vm.ID)

		for _, excludedId := range excludeInstanceIds {
			if instanceId == excludedId {
				logger.Debugf("Instance %s is chosen for termination, no need to protect", instanceId)
				continue
			}
		}
		if !protected {
			vmInfo := VmInfo{HostName: *vm.Properties.OSProfile.ComputerName, InstanceId: instanceId}
			vmsWithoutProtection = append(vmsWithoutProtection, vmInfo)
		}
	}
	logger.Info().Msgf("%d VMs are not protected from deletion", len(vmsWithoutProtection))

	// set deletion protection in case it's not set
	for _, vm := range vmsWithoutProtection {
		logger.Info().Msgf("Setting deletion protection for VM %v", vm)
		// do not retry, but report
		common.RetrySetDeletionProtectionAndReport(ctx, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, vmScaleSetName, vm.InstanceId, vm.HostName, 0, time.Second)
	}
}

func Terminate(ctx context.Context, scaleResponse protocol.ScaleResponse, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName string) (response protocol.TerminatedInstancesResponse, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("Running termination function...")

	response.Version = protocol.Version

	if scaleResponse.Version != protocol.Version {
		err = errors.New("incompatible scale response version")
		return
	}

	if vmScaleSetName == "" {
		err = errors.New("instance group is mandatory")
		return
	}
	if len(scaleResponse.Hosts) == 0 {
		err = errors.New("hosts list must not be empty")
		return
	}

	response.TransientErrors = scaleResponse.TransientErrors[0:len(scaleResponse.TransientErrors):len(scaleResponse.TransientErrors)]

	// get VMs expanded list which will be used later
	vms, err := getScaleSetVmsExpandedView(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		err = fmt.Errorf("cannot get VMs list for vmss %s: %v", vmScaleSetName, err)
		return
	}

	unhealthyInstanceIds := getUnhealthyInstancesToTerminate(ctx, vms)
	errs := terminateUnhealthyInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, unhealthyInstanceIds)
	response.AddTransientErrors(errs)

	logger.Info().Msgf("Instances set for explicit removal: %s", scaleResponse.ToTerminate)
	deltaInstanceIds, err := getDeltaInstancesIds(ctx, subscriptionId, resourceGroupName, vmScaleSetName, scaleResponse)
	if err != nil {
		logger.Error().Msgf("%s", err)
		return
	}

	// NOTE: we want to have deletion protection set for all instances (which are not selected for termination)
	// in order to avoid races, this step is presented here, during "terminate" step
	unprotectedVmIds := append(deltaInstanceIds, unhealthyInstanceIds...)
	setDeletionProtection(ctx, vms, unprotectedVmIds, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName)

	if len(deltaInstanceIds) == 0 {
		logger.Info().Msgf("No delta instances ids")
		return
	}

	candidatesToTerminate, err := common.FilterSpecificScaleSetInstances(ctx, vms, deltaInstanceIds)
	if err != nil {
		logger.Error().Msgf("%s", err)
		return
	}

	terminatedInstancesMap, errs := terminateUnneededInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, candidatesToTerminate, scaleResponse.ToTerminate)
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

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	vmScaleSetName := fmt.Sprintf("%s-%s-vmss", prefix, clusterName)

	var scaleResponse protocol.ScaleResponse
	err := json.NewDecoder(r.Body).Decode(&scaleResponse)
	if err != nil {
		logger.Error().Msg("Bad request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	terminateResponse, err := Terminate(ctx, scaleResponse, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName)
	if err != nil {
		logger.Error().Err(err).Send()
		common.RespondWithError(w, err, http.StatusInternalServerError)
		return
	}

	common.RespondWithJson(w, terminateResponse, http.StatusOK)
}
