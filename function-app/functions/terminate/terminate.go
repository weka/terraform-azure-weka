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
	"weka-deployment/protocol"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
)

type instancesMap map[string]*armcompute.VirtualMachineScaleSetVM

func instancesToMap(instances []*armcompute.VirtualMachineScaleSetVM) instancesMap {
	im := make(instancesMap)
	for _, instance := range instances {
		im[common.GetScaleSetVmId(*instance.ID)] = instance
	}
	return im
}

func getDeltaInstancesIds(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string, scaleResponse protocol.ScaleResponse) (deltaInstanceIDs []string, err error) {
	logger := common.LoggerFromCtx(ctx)
	logger.Info().Msg("Getting delta instances")
	netInterfaces, err := common.GetScaleSetVmsNetworkInterfaces(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
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
	logger := common.LoggerFromCtx(ctx)

	terminateInstanceIds := make([]string, 0, 0)
	imap := instancesToMap(instances)

	for _, instance := range instances {
		logger.Info().Msgf("Handling instance %s(%s) removal", *instance.Name, *instance.InstanceID)
		if !setForExplicitRemoval(instance, explicitRemoval) {
			instanceCreationTime := getInstanceCreationTime(instance)
			if instanceCreationTime == nil {
				logger.Info().Msgf("Couldn't retrieve instance %s creation time, it is probably too new, giving grace time before removal", *instance.InstanceID)
				continue
			}
			if time.Now().Sub(*instanceCreationTime) < time.Minute*30 {
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

func terminateUnhealthyInstances(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string) (errs []error) {
	logger := common.LoggerFromCtx(ctx)
	var toTerminate []string

	expand := "instanceView"
	vms, err := common.GetScaleSetInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, &expand)
	if err != nil {
		errs = append(errs, err)
		return
	}

	for _, vm := range vms {
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
	_, terminateErrors := common.TerminateScaleSetInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, toTerminate)
	errs = append(errs, terminateErrors...)

	return
}

func Terminate(ctx context.Context, scaleResponse protocol.ScaleResponse, subscriptionId, resourceGroupName, vmScaleSetName string) (response protocol.TerminatedInstancesResponse, err error) {
	logger := common.LoggerFromCtx(ctx)
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

	errs := terminateUnhealthyInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if len(errs) != 0 {
		response.AddTransientErrors(errs)
	}

	logger.Info().Msgf("Instances set for explicit removal: %s", scaleResponse.ToTerminate)
	deltaInstanceIds, err := getDeltaInstancesIds(ctx, subscriptionId, resourceGroupName, vmScaleSetName, scaleResponse)
	if err != nil {
		logger.Error().Msgf("%s", err)
		return
	}

	if len(deltaInstanceIds) == 0 {
		logger.Info().Msgf("No delta instances ids")
		return
	}

	expand := "instanceView"
	candidatesToTerminate, err := common.GetSpecificScaleSetInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, deltaInstanceIds, &expand)
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

	ctx := r.Context()
	logger := common.LoggerFromCtx(ctx)

	vmScaleSetName := fmt.Sprintf("%s-%s-vmss", prefix, clusterName)

	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})
	var invokeRequest common.InvokeRequest

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

	var scaleResponse protocol.ScaleResponse

	if json.Unmarshal([]byte(reqData["Body"].(string)), &scaleResponse) != nil {
		logger.Error().Msgf("Failed to parse scaleResponse:%s", reqData["Body"].(string))
		return
	}

	terminateResponse, err := Terminate(ctx, scaleResponse, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		resData["body"] = err.Error()
	} else {
		resData["body"] = terminateResponse
	}
	outputs["res"] = resData
	invokeResponse := common.InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}
