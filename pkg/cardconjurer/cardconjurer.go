package cardconjurer

import (
	"cardconjurer-automation/pkg/common"
	"context"
	"errors"
	"go.uber.org/zap"
	"sync"
)

type CardConjurer struct {
	config     *Config
	cards      []common.CardInfo
	cardsChan  chan common.CardInfo
	outputChan chan common.CardInfo
	logger     *zap.SugaredLogger
}

func New(cfg *Config, logger *zap.SugaredLogger, cards []common.CardInfo) (*CardConjurer, error) {

	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	return &CardConjurer{
		config:     cfg,
		cards:      cards,
		outputChan: make(chan common.CardInfo, 1000),
		logger:     logger.With("module", "CardConjurer"),
	}, nil
}

func (cc *CardConjurer) GetOutputChan() <-chan common.CardInfo {
	return cc.outputChan
}

func (cc *CardConjurer) ListCards() {
	for _, card := range cc.cards {
		cc.logger.Info(card.GetFullName())
	}
}

func (cc *CardConjurer) Run(ctx context.Context) {
	var wg sync.WaitGroup
	cc.cardsChan = make(chan common.CardInfo, cc.config.Workers)

	cc.logger.Infof("Starting %d worker(s)...", cc.config.Workers)
	// Start workers
	for i := 0; i < cc.config.Workers; i++ {
		wg.Add(1)
		go cc.startWorker(i, ctx, &wg)
	}

	cc.logger.Infof("Sending %d cards to workers...", len(cc.cards))
	// Send cards to the channel
	for _, card := range cc.cards {
		select {
		case <-ctx.Done():
			cc.logger.Info("Context cancelled, closing cardsChan and waiting for workers...")
			close(cc.cardsChan)
			wg.Wait()
			cc.logger.Info("All workers finished.")
			return
		case cc.cardsChan <- card:
			cc.logger.Infof("Card '%s' sent to worker.", card.GetFullName())
		}
	}
	cc.logger.Info("All cards have been sent to workers. Closing cardsChan.")
	close(cc.cardsChan)
	wg.Wait()
	close(cc.outputChan)
	cc.logger.Info("All workers have finished their work.")
}

func (cc *CardConjurer) startWorker(id int, ctx context.Context, wg *sync.WaitGroup) {
	cc.logger.Infof("Starting worker %d", id)

	defer func() {
		cc.logger.Infof("Worker %d: finished", id)
		wg.Done()
	}()

	w := newWorker(id, cc.logger, cc.config)
	w.startWorker(ctx, cc.cardsChan, cc.outputChan)
}
