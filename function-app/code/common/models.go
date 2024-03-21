package common

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
)

type ScaleSetParams struct {
	SubscriptionId    string
	ResourceGroupName string
	ScaleSetName      string
	Flexible          bool
}

// General struct representing VM data relevant both for VMs and VMSS VMs models
type VMInfoSummary struct {
	ID                   string
	InstanceID           string
	Name                 string
	ProvisioningState    *string
	ComputerName         *string
	NetworkProfile       *armcompute.NetworkProfile
	VMHealth             *armcompute.VirtualMachineHealthStatus
	InstanceViewStatuses []*armcompute.InstanceViewStatus
	ProtectionPolicy     *armcompute.VirtualMachineScaleSetVMProtectionPolicy
}

func UniformVmssVMsToVmInfoSummary(vms []*armcompute.VirtualMachineScaleSetVM) []*VMInfoSummary {
	res := make([]*VMInfoSummary, len(vms))
	for i, vm := range vms {
		res[i] = &VMInfoSummary{
			ID:                *vm.ID,
			InstanceID:        *vm.InstanceID,
			Name:              *vm.Name,
			ProvisioningState: vm.Properties.ProvisioningState,
			NetworkProfile:    vm.Properties.NetworkProfile,
			ProtectionPolicy:  vm.Properties.ProtectionPolicy,
		}
		if vm.Properties.InstanceView != nil {
			res[i].ComputerName = vm.Properties.InstanceView.ComputerName
			res[i].VMHealth = vm.Properties.InstanceView.VMHealth
			res[i].InstanceViewStatuses = vm.Properties.InstanceView.Statuses
		}

	}
	return res
}

func VMsToVmInfoSummary(vms []*armcompute.VirtualMachine) []*VMInfoSummary {
	res := make([]*VMInfoSummary, len(vms))
	for i, vm := range vms {
		res[i] = &VMInfoSummary{
			ID:                *vm.ID,
			InstanceID:        *vm.Name,
			Name:              *vm.Name,
			ProvisioningState: vm.Properties.ProvisioningState,
			NetworkProfile:    vm.Properties.NetworkProfile,
		}
		if vm.Properties.InstanceView != nil {
			res[i].ComputerName = vm.Properties.InstanceView.ComputerName
			res[i].VMHealth = vm.Properties.InstanceView.VMHealth
			res[i].InstanceViewStatuses = vm.Properties.InstanceView.Statuses
		}
	}
	return res
}
