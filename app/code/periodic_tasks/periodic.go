package periodic

import (
	"context"
	"fmt"
	"time"

	"github.com/weka/go-cloud-lib/logging"
)

type PeriodicTask struct {
	name        string
	runInterval time.Duration
	closing     chan chan bool
}

func NewPeriodicTask(name string, runInterval time.Duration) PeriodicTask {
	p := PeriodicTask{
		name:        name,
		runInterval: runInterval,
	}
	p.closing = make(chan chan bool)
	return p
}

func (p *PeriodicTask) run(ctx context.Context, workFunc func(ctx context.Context) error) {
	ticker := time.NewTicker(p.runInterval)
	logger := logging.LoggerFromCtx(ctx)

	for {
		select {
		case errc := <-p.closing:
			ticker.Stop()
			logger.Info().Msgf("Stopping %s...", p.name)
			errc <- true
			return
		case <-ticker.C:
			logger.Debug().Msgf("Start running %s", p.name)
			err := workFunc(ctx)
			if err != nil {
				err := fmt.Errorf("failed to run %s: %v", p.name, err)
				logger.Warn().Err(err).Send()
			}
			logger.Debug().Msgf("Finished running %s", p.name)
		}
	}
}

func (p *PeriodicTask) Stop() bool {
	errc := make(chan bool)
	p.closing <- errc
	return <-errc
}
