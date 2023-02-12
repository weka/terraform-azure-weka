package terminate

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"time"
	"weka-deployment/common"
	"weka-deployment/protocol"
)

type instancesMap map[string]*armcompute.VirtualMachineScaleSetVM

func instancesToMap(instances []*armcompute.VirtualMachineScaleSetVM) instancesMap {
	im := make(instancesMap)
	for _, instance := range instances {
		im[*instance.Name] = instance
	}
	return im
}

func getDeltaInstancesIds(subscriptionId, resourceGroupName, vmScaleSetName string, asgInstanceIds []string, scaleResponse protocol.ScaleResponse) (deltaInstanceIDs []string, err error) {
	log.Info().Msg("Getting delta instances")
	netInterfaces, err := common.GetScaleSetVmsNetworkInterfaces(subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		return
	}
	instanceIdPrivateIp := map[string]string{}

	for _, ni := range netInterfaces {
		id := common.GetScaleSetVmId(*ni.Properties.VirtualMachine.ID)
		privateIp := *ni.Properties.IPConfigurations[0].Properties.PrivateIPAddress
		instanceIdPrivateIp[id] = privateIp
	}

	instanceIdsSet := common.GetInstanceIdsSet(scaleResponse)

	for _, id := range asgInstanceIds {
		log.Info().Msgf("Checking machine id:%s", id)
		if _, ok := instanceIdsSet[id]; !ok {
			deltaInstanceIDs = append(deltaInstanceIDs, id)
		}
	}
	log.Info().Msgf("Delta instances%s", deltaInstanceIDs)
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

func terminateUnneededInstances(subscriptionId, resourceGroupName, vmScaleSetName string, instances []*armcompute.VirtualMachineScaleSetVM, explicitRemoval []protocol.HgInstance) (terminated []*armcompute.VirtualMachineScaleSetVM, errs []error) {
	terminateInstanceIds := make([]string, 0, 0)
	imap := instancesToMap(instances)

	for _, instance := range instances {
		statuses := instance.Properties.InstanceView.Statuses
		firstStatus := statuses[0]
		lastStatus := statuses[len(statuses)-1]
		if !setForExplicitRemoval(instance, explicitRemoval) {
			if time.Now().Sub(*firstStatus.Time) < time.Minute*30 {
				continue
			}
		}
		instanceState := *lastStatus.Code
		if instanceState != "STOPPING" && instanceState != "TERMINATED" {
			terminateInstanceIds = append(terminateInstanceIds, *instance.Name)
		}
	}

	terminatedInstances, errs := common.TerminateSclaeSetInstances(subscriptionId, resourceGroupName, vmScaleSetName, terminateInstanceIds)

	for _, id := range terminatedInstances {
		terminated = append(terminated, imap[id])
	}
	return
}

func terminateUnhealthyInstances(subscriptionId, resourceGroupName, vmScaleSetName string) (errs []error) {
	var toTerminate []string

	expand := "instanceView"
	vms, err := common.GetScaleSetInstances(subscriptionId, resourceGroupName, vmScaleSetName, &expand)
	if err != nil {
		errs = append(errs, err)
		return
	}

	for _, vm := range vms {
		healthStatus := *vm.Properties.InstanceView.VMHealth.Status.Code
		log.Info().Msgf("handling instance %s(%s)", *vm.Name, healthStatus)
		if healthStatus == "UNHEALTHY" {
			statuses := vm.Properties.InstanceView.Statuses
			status := *statuses[len(statuses)-1].Code
			log.Debug().Msgf("instance state: %s", status)
			if status == "STOPPED" {
				toTerminate = append(toTerminate, common.GetScaleSetVmId(*vm.ID))
			}

		}
	}

	log.Debug().Msgf("found %d stopped instances", len(toTerminate))
	_, terminateErrors := common.TerminateSclaeSetInstances(subscriptionId, resourceGroupName, vmScaleSetName, toTerminate)
	errs = append(errs, terminateErrors...)

	return
}

func Terminate(scaleResponse protocol.ScaleResponse, subscriptionId, resourceGroupName, vmScaleSetName string) (response protocol.TerminatedInstancesResponse, err error) {
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

	asgInstanceIds, err := common.GetScaleSetInstanceIds(subscriptionId, resourceGroupName, vmScaleSetName)
	log.Info().Msgf("Found %d instances on ASG", len(asgInstanceIds))
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	errs := terminateUnhealthyInstances(subscriptionId, resourceGroupName, vmScaleSetName)
	if len(errs) != 0 {
		response.AddTransientErrors(errs)
	}

	deltaInstanceIds, err := getDeltaInstancesIds(subscriptionId, resourceGroupName, vmScaleSetName, asgInstanceIds, scaleResponse)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	if len(deltaInstanceIds) == 0 {
		log.Info().Msgf("No delta instances ids")
		return
	}

	candidatesToTerminate, err := common.GetSpecificScaleSetInstances(subscriptionId, resourceGroupName, vmScaleSetName, deltaInstanceIds)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	terminatedInstances, errs := terminateUnneededInstances(subscriptionId, resourceGroupName, vmScaleSetName, candidatesToTerminate, scaleResponse.ToTerminate)
	response.AddTransientErrors(errs)

	for _, instance := range terminatedInstances {
		response.Instances = append(response.Instances, protocol.TerminatedInstance{
			InstanceId: common.GetScaleSetVmId(*instance.Name),
			Creation:   *instance.Properties.InstanceView.Statuses[0].Time,
		})
	}

	return
}

func Handler(w http.ResponseWriter, r *http.Request) {
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")

	vmScaleSetName := fmt.Sprintf("%s-%s-vmss", prefix, clusterName)

	outputs := make(map[string]interface{})
	resData := make(map[string]interface{})
	var invokeRequest common.InvokeRequest

	d := json.NewDecoder(r.Body)
	err := d.Decode(&invokeRequest)
	if err != nil {
		log.Error().Msg("Bad request")
		return
	}

	var reqData map[string]interface{}
	err = json.Unmarshal(invokeRequest.Data["req"], &reqData)
	if err != nil {
		log.Error().Msg("Bad request")
		return
	}

	var scaleResponse protocol.ScaleResponse

	if json.Unmarshal([]byte(reqData["Body"].(string)), &scaleResponse) != nil {
		log.Error().Msg("Bad request")
		return
	}

	terminateResponse, err := Terminate(scaleResponse, subscriptionId, resourceGroupName, vmScaleSetName)
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
