package periodic

import (
	"context"
	"fmt"
	"os"
	"time"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/logging"
)

type ScaleUpPeriodicTask struct {
	PeriodicTask
}

func NewScaleUpPeriodicTask(interval time.Duration) ScaleUpPeriodicTask {
	return ScaleUpPeriodicTask{NewPeriodicTask("scale_up", interval)}
}

func (p *ScaleUpPeriodicTask) Run(ctx context.Context) {
	p.run(ctx, p.ScaleUpWorkflow)
}

func (p *ScaleUpPeriodicTask) ScaleUpWorkflow(ctx context.Context) error {
	logger := logging.LoggerFromCtx(ctx)

	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	state, err := common.ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		err := fmt.Errorf("cannot read state: %v", err)
		return err
	}

	if !state.Clusterized {
		msg := "Not clusterized yet, skipping..."
		logger.Info().Msg(msg)
		return nil
	}

	err = common.UpdateVmScaleSetNum(ctx, subscriptionId, resourceGroupName, vmScaleSetName, int64(state.DesiredSize))
	if err != nil {
		err := fmt.Errorf("cannot update scale set size: %v", err)
		return err
	}

	msg := fmt.Sprintf("updated size to %d successfully", state.DesiredSize)
	logger.Info().Msg(msg)
	return nil
}
