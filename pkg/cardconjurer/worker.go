package cardconjurer

import (
	"cardconjurer-automation/pkg/common"
	"context"
	"fmt"
	"log"
	"time"
)

type worker struct {
	workerID    int
	config      *Config
	tempDirName string
}

func newWorker(workerID int, config *Config) *worker {
	return &worker{
		workerID:    workerID,
		config:      config,
		tempDirName: fmt.Sprintf("%s_%d", config.ProjectName, workerID),
	}
}

func (w *worker) startWorker(ctx context.Context, cardsChan <-chan common.CardInfo, outputChan chan<- common.CardInfo) {
	browserCtx, err := w.openBrowser(ctx)
	if err != nil {
		log.Printf("Worker %d: Error opening browser: %v", w.workerID, err)
		return
	}

	defer func() {
		w.closeBrowser(browserCtx)
		log.Printf("Worker %d: Browser closed", w.workerID)
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d: Context finished, stopping worker", w.workerID)
			return
		case card, ok := <-cardsChan:
			if !ok {
				log.Printf("Worker %d: cardsChan closed, stopping worker", w.workerID)
				return
			}

			log.Printf("Worker %d: Processing card: %s", w.workerID, card.GetFullName())

			err := w.handleCard(card, browserCtx)
			if err != nil {
				continue
			}

			outputChan <- card
			time.Sleep(time.Millisecond * 250)

			log.Printf("Worker %d: Card '%s' processed.", w.workerID, card.GetFullName())
		}
	}
}

func (w *worker) handleCard(card common.CardInfo, browserCtx context.Context) error {
	err := w.importCard(card, browserCtx)
	if err != nil {
		log.Printf("Worker %d: Error importing card '%s': %v", w.workerID, card.GetFullName(), err)
		return err
	}

	log.Printf("Worker %d: Card '%s' imported, adding margin...", w.workerID, card.GetFullName())
	err = w.addMargin(browserCtx)
	if err != nil {
		log.Printf("Worker %d: Error adding margin for card '%s': %v", w.workerID, card.GetFullName(), err)
		return err
	}

	err = w.replaceArtwork(card, browserCtx)
	if err != nil {
		log.Printf("Worker %d: Error replacing artwork for card '%s': %v", w.workerID, card.GetFullName(), err)
		return err
	}

	err = w.removeSetSymbol(browserCtx)
	if err != nil {
		log.Printf("Worker %d: Error removing set symbol for card '%s': %v", w.workerID, card.GetFullName(), err)
		return err
	}

	err = w.saveCard(card, browserCtx)
	if err != nil {
		log.Printf("Worker %d: Error saving card '%s': %v", w.workerID, card.GetFullName(), err)
		return err
	}

	return nil
}
