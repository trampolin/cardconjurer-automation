package cardconjurer

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

type CardConjurer struct {
	config    *Config
	cards     []CardInfo
	cardsChan chan CardInfo
}

func New(cfg *Config, cards []CardInfo) (*CardConjurer, error) {

	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	return &CardConjurer{
		config: cfg,
		cards:  cards,
	}, nil
}

func (cc *CardConjurer) ListCards() {
	for _, card := range cc.cards {
		log.Println(card.GetFullName())
	}
}

func (cc *CardConjurer) Run(ctx context.Context) {
	var wg sync.WaitGroup
	cc.cardsChan = make(chan CardInfo, cc.config.Workers)

	log.Printf("Starte %d Worker...", cc.config.Workers)
	// Worker starten
	for i := 0; i < cc.config.Workers; i++ {
		wg.Add(1)
		go cc.startWorker(i, ctx, &wg)
	}

	log.Printf("Sende %d Karten an die Worker...", len(cc.cards))
	// Karten in den Channel senden
	for _, card := range cc.cards {
		select {
		case <-ctx.Done():
			log.Println("Context abgebrochen, schließe cardsChan und warte auf Worker...")
			close(cc.cardsChan)
			wg.Wait()
			log.Println("Alle Worker beendet.")
			return
		case cc.cardsChan <- card:
			log.Printf("Karte '%s' an Worker gesendet.", card.GetFullName())
		}
	}
	log.Println("Alle Karten wurden an die Worker gesendet. Schließe cardsChan.")
	close(cc.cardsChan)
	wg.Wait()
	log.Println("Alle Worker haben ihre Arbeit beendet.")
}

func (cc *CardConjurer) startWorker(id int, ctx context.Context, wg *sync.WaitGroup) {
	log.Printf("Starte Worker %d", id)

	defer func() {
		log.Printf("Worker %d: beendet", id)
		wg.Done()
	}()

	browserCtx, err := cc.openBrowser(ctx)
	if err != nil {
		log.Printf("Worker %d: Fehler beim Öffnen des Browsers: %v", id, err)
		return
	}

	defer func() {
		cc.closeBrowser(browserCtx)
		log.Printf("Worker %d: Browser geschlossen", id)
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d: Context beendet, Worker wird gestoppt", id)
			return
		case card, ok := <-cc.cardsChan:
			if !ok {
				log.Printf("Worker %d: cardsChan geschlossen, Worker wird gestoppt", id)
				return
			}

			log.Printf("Worker %d: Bearbeite Karte: %s", id, card.GetFullName())

			err := cc.importCard(card, browserCtx)
			if err != nil {
				log.Printf("Worker %d: Fehler beim Importieren der Karte '%s': %v", id, card.GetFullName(), err)
				continue
			}

			log.Printf("Worker %d: Karte '%s' importiert, füge Margin hinzu...", id, card.GetFullName())
			err = cc.addMargin(browserCtx)
			if err != nil {
				log.Printf("Worker %d: Fehler beim Hinzufügen von Margin für Karte '%s': %v", id, card.GetFullName(), err)
				continue
			}

			err = cc.replaceArtwork(card, browserCtx)
			if err != nil {
				log.Printf("Worker %d: Fehler beim Ersetzen des Artwork für Karte '%s': %v", id, card.GetFullName(), err)
				continue
			}

			err = cc.removeSetSymbol(browserCtx)
			if err != nil {
				log.Printf("Worker %d: Fehler beim Entfernen des Set-Symbols für Karte '%s': %v", id, card.GetFullName(), err)
				continue
			}

			err = cc.saveCard(card, browserCtx)
			if err != nil {
				log.Printf("Worker %d: Fehler beim Speichern der Karte '%s': %v", id, card.GetFullName(), err)
				continue
			}

			time.Sleep(time.Millisecond * 250)

			log.Printf("Worker %d: Karte '%s' fertig verarbeitet.", id, card.GetFullName())
		}
	}
}
