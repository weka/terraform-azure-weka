package main

import (
	"context"
	"strings"
	"testing"
	"weka-deployment/common"
)

func Test_fetchPrivateIps(t *testing.T) {
	subscriptionId := "d2f248b9-d054-477f-b7e8-413921532c2a"
	resourceGroupName := "jassaf-rg"
	vmScaleSetName := "jassaf-poc-vmss"
	ctx := context.TODO()

	vmssParams := &common.ScaleSetParams{
		SubscriptionId:    subscriptionId,
		ResourceGroupName: resourceGroupName,
		ScaleSetName:      vmScaleSetName,
		Flexible:          false,
	}

	vmsPrivateIps, err := common.GetVmsPrivateIps(ctx, vmssParams)
	if err != nil {
		t.Logf("failed fetching private ips: %s", err)
		return
	}

	t.Logf("ips map: %s", vmsPrivateIps)

	var ipsList []string
	instances := []string{
		"jassaf-poc-vmss_6",
		"jassaf-poc-vmss_4",
		"jassaf-poc-vmss_3",
		"jassaf-poc-vmss_0",
		"jassaf-poc-vmss_1",
		"jassaf-poc-vmss_5",
	}

	for _, instance := range instances {
		ipsList = append(ipsList, vmsPrivateIps[instance])
	}
	ips := strings.Join(ipsList, ",")
	t.Logf("ips: %s", ips)

}
