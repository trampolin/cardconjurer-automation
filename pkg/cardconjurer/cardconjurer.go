package cardconjurer

import (
	"cardconjurer-automation/pkg/common"
	"context"
	"errors"
	"log"
	"sync"
)

type CardConjurer struct {
	config     *Config
	cards      []common.CardInfo
	cardsChan  chan common.CardInfo
	outputChan chan common.CardInfo
}

func New(cfg *Config, cards []common.CardInfo) (*CardConjurer, error) {

	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	return &CardConjurer{
		config:     cfg,
		cards:      cards,
		outputChan: make(chan common.CardInfo, 1000),
	}, nil
}

func (cc *CardConjurer) GetOutputChan() <-chan common.CardInfo {
	return cc.outputChan
}

func (cc *CardConjurer) ListCards() {
	for _, card := range cc.cards {
		log.Println(card.GetFullName())
	}
}

func (cc *CardConjurer) Run(ctx context.Context) {
	var wg sync.WaitGroup
	cc.cardsChan = make(chan common.CardInfo, cc.config.Workers)

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

	w := newWorker(id, cc.config)
	w.startWorker(ctx, cc.cardsChan, cc.outputChan)
}
