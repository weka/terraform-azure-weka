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

func (p *PeriodicTask) ScaleUp(ctx context.Context) error {
	return nil
}

func (p *PeriodicTask) Run(ctx context.Context) {
	p.run(ctx, p.ScaleUp)
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
			logger.Info().Msgf("Start running %s", p.name)
			err := workFunc(ctx)
			if err != nil {
				err := fmt.Errorf("failed to run %s: %v", p.name, err)
				logger.Warn().Err(err).Send()
			}
			logger.Info().Msgf("Finished running %s", p.name)
		}
	}
}

func (p *PeriodicTask) Stop() bool {
	errc := make(chan bool)
	p.closing <- errc
	return <-errc
}
