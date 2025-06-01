package cardconjurer

import (
	"cardconjurer-automation/pkg/common"
	"context"
	"fmt"
	"go.uber.org/zap"
	"time"
)

type worker struct {
	workerID    int
	config      *Config
	tempDirName string
	logger      *zap.SugaredLogger
}

func newWorker(workerID int, logger *zap.SugaredLogger, config *Config) *worker {
	return &worker{
		workerID:    workerID,
		config:      config,
		tempDirName: fmt.Sprintf("%s_%d", config.ProjectName, workerID),
		logger:      logger.With("worker_id", workerID),
	}
}

func (w *worker) startWorker(ctx context.Context, cardsChan <-chan common.CardInfo, outputChan chan<- common.CardInfo) {
	browserCtx, err := w.openBrowser(ctx)
	if err != nil {
		w.logger.Errorf("Error opening browser: %v", err)
		return
	}

	defer func() {
		w.closeBrowser(browserCtx)
		w.logger.Info("Browser closed")
	}()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Context finished, stopping worker")
			return
		case card, ok := <-cardsChan:
			if !ok {
				w.logger.Info("cardsChan closed, stopping worker")
				return
			}

			w.logger.Infof("Processing card: %s", card.GetFullName())

			err := w.handleCard(card, browserCtx)
			if err != nil {
				continue
			}

			outputChan <- card
			time.Sleep(time.Millisecond * 250)

			w.logger.Infof("Card '%s' processed.", card.GetFullName())
		}
	}
}

func (w *worker) handleCard(card common.CardInfo, browserCtx context.Context) error {
	err := w.importCard(card, browserCtx)
	if err != nil {
		w.logger.Errorf("Error importing card '%s': %v", card.GetFullName(), err)
		return err
	}

	w.logger.Infof("Card '%s' imported, adding margin...", card.GetFullName())
	err = w.addMargin(browserCtx)
	if err != nil {
		w.logger.Errorf("Error adding margin for card '%s': %v", card.GetFullName(), err)
		return err
	}

	err = w.replaceArtwork(card, browserCtx)
	if err != nil {
		w.logger.Errorf("Error replacing artwork for card '%s': %v", card.GetFullName(), err)
		return err
	}

	err = w.removeSetSymbol(browserCtx)
	if err != nil {
		w.logger.Errorf("Error removing set symbol for card '%s': %v", card.GetFullName(), err)
		return err
	}

	err = w.saveCard(card, browserCtx)
	if err != nil {
		w.logger.Errorf("Error saving card '%s': %v", card.GetFullName(), err)
		return err
	}

	return nil
}
