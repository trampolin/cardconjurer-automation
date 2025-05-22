package cardconjurer

import (
	"context"
	"log"
	"time"
)

type worker struct {
	workerID int
	config   *Config
}

func newWorker(workerID int, config *Config) *worker {
	return &worker{
		workerID: workerID,
		config:   config,
	}
}

func (w *worker) startWorker(ctx context.Context, cardsChan <-chan CardInfo) {
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

			err := w.importCard(card, browserCtx)
			if err != nil {
				log.Printf("Worker %d: Fehler beim Importieren der Karte '%s': %v", w.workerID, card.GetFullName(), err)
				continue
			}

			log.Printf("Worker %d: Karte '%s' importiert, füge Margin hinzu...", w.workerID, card.GetFullName())
			err = w.addMargin(browserCtx)
			if err != nil {
				log.Printf("Worker %d: Fehler beim Hinzufügen von Margin für Karte '%s': %v", w.workerID, card.GetFullName(), err)
				continue
			}

			err = w.replaceArtwork(card, browserCtx)
			if err != nil {
				log.Printf("Worker %d: Fehler beim Ersetzen des Artwork für Karte '%s': %v", w.workerID, card.GetFullName(), err)
				continue
			}

			err = w.removeSetSymbol(browserCtx)
			if err != nil {
				log.Printf("Worker %d: Fehler beim Entfernen des Set-Symbols für Karte '%s': %v", w.workerID, card.GetFullName(), err)
				continue
			}

			err = w.saveCard(card, browserCtx)
			if err != nil {
				log.Printf("Worker %d: Fehler beim Speichern der Karte '%s': %v", w.workerID, card.GetFullName(), err)
				continue
			}

			time.Sleep(time.Millisecond * 250)

			log.Printf("Worker %d: Karte '%s' fertig verarbeitet.", w.workerID, card.GetFullName())
		}
	}
}
