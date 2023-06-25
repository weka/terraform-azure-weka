package periodic

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"weka-deployment/common"
	"weka-deployment/functions/fetch"
	"weka-deployment/functions/terminate"

	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/scale_down"
)

type ScaleDownPeriodicTask struct {
	PeriodicTask
}

func NewScaleDownPeriodicTask(interval time.Duration) ScaleDownPeriodicTask {
	return ScaleDownPeriodicTask{NewPeriodicTask("scale_down", interval)}
}

func (p *ScaleDownPeriodicTask) Run(ctx context.Context) {
	p.run(ctx, p.ScaleDownWorkflow)
}

func (p *ScaleDownPeriodicTask) ScaleDownWorkflow(ctx context.Context) error {
	logger := logging.LoggerFromCtx(ctx)

	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	info, err := fetch.GetScaleSetInfoResponse(
		ctx, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName, keyVaultUri,
	)
	if err != nil {
		err := fmt.Errorf("cannot 'fetch' instances: %v", err)
		return err
	}

	scaleResponse, err := scale_down.ScaleDown(ctx, info)
	if err != nil {
		err := fmt.Errorf("error while running 'scale_down': %v", err)
		return err
	}

	terminateResponse, err := terminate.Terminate(ctx, scaleResponse, subscriptionId, resourceGroupName, vmScaleSetName, stateContainerName, stateStorageName)
	if err != nil {
		err := fmt.Errorf("error while running 'terminate': %v", err)
		return err
	}

	logger.Debug().Msgf("terminate output: %#v", terminateResponse)
	errs := terminateResponse.TransientErrors
	if len(errs) == 0 {
		msg := "no transient errors found"
		logger.Debug().Msg(msg)
		return nil
	}

	logger.Error().Msgf("transient errors: %s", errs)
	err = fmt.Errorf(strings.Join(errs, "; "))
	return err
}
