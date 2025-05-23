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
		log.Printf("Worker %d: Fehler beim Öffnen des Browsers: %v", w.workerID, err)
		return
	}

	defer func() {
		w.closeBrowser(browserCtx)
		log.Printf("Worker %d: Browser geschlossen", w.workerID)
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d: Context beendet, Worker wird gestoppt", w.workerID)
			return
		case card, ok := <-cardsChan:
			if !ok {
				log.Printf("Worker %d: cardsChan geschlossen, Worker wird gestoppt", w.workerID)
				return
			}

			log.Printf("Worker %d: Bearbeite Karte: %s", w.workerID, card.GetFullName())

			if false {
				err := w.handleCard(card, browserCtx)
				if err != nil {
					continue
				}
			}

			outputChan <- card
			time.Sleep(time.Millisecond * 250)

			log.Printf("Worker %d: Karte '%s' fertig verarbeitet.", w.workerID, card.GetFullName())
		}
	}
}

func (w *worker) handleCard(card common.CardInfo, browserCtx context.Context) error {
	err := w.importCard(card, browserCtx)
	if err != nil {
		log.Printf("Worker %d: Fehler beim Importieren der Karte '%s': %v", w.workerID, card.GetFullName(), err)
		return err
	}

	log.Printf("Worker %d: Karte '%s' importiert, füge Margin hinzu...", w.workerID, card.GetFullName())
	err = w.addMargin(browserCtx)
	if err != nil {
		log.Printf("Worker %d: Fehler beim Hinzufügen von Margin für Karte '%s': %v", w.workerID, card.GetFullName(), err)
		return err
	}

	err = w.replaceArtwork(card, browserCtx)
	if err != nil {
		log.Printf("Worker %d: Fehler beim Ersetzen des Artwork für Karte '%s': %v", w.workerID, card.GetFullName(), err)
		return err
	}

	err = w.removeSetSymbol(browserCtx)
	if err != nil {
		log.Printf("Worker %d: Fehler beim Entfernen des Set-Symbols für Karte '%s': %v", w.workerID, card.GetFullName(), err)
		return err
	}

	err = w.saveCard(card, browserCtx)
	if err != nil {
		log.Printf("Worker %d: Fehler beim Speichern der Karte '%s': %v", w.workerID, card.GetFullName(), err)
		return err
	}

	return nil
}
