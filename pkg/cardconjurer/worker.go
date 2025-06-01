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
		w.logger.Errorw("Error opening browser", "error", err)
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

			err := w.handleCard(card, browserCtx)
			if err != nil {
				continue
			}

			outputChan <- card
			time.Sleep(time.Millisecond * 250)

			w.logger.Info("Card processed.")
		}
	}
}

func (w *worker) handleCard(card common.CardInfo, browserCtx context.Context) error {
	oldLogger := w.logger
	w.logger = w.logger.With("card", card.GetFullName())
	defer func() {
		w.logger = oldLogger
	}()

	w.logger.Info("Processing card")

	err := w.importCard(card, browserCtx)
	if err != nil {
		w.logger.Errorw("Error importing card", "error", err)
		return err
	}

	w.logger.Info("Card imported, adding margin")
	err = w.addMargin(browserCtx)
	if err != nil {
		w.logger.Errorw("Error adding margin", "error", err)
		return err
	}

	err = w.replaceArtwork(card, browserCtx)
	if err != nil {
		w.logger.Errorw("Error replacing artwork", "error", err)
		return err
	}

	err = w.removeSetSymbol(browserCtx)
	if err != nil {
		w.logger.Errorw("Error removing set symbol", "error", err)
		return err
	}

	err = w.saveCard(card, browserCtx)
	if err != nil {
		w.logger.Errorw("Error saving card", "error", err)
		return err
	}

	return nil
}
