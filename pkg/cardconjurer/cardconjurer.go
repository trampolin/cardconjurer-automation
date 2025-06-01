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

	log.Printf("Starting %d worker(s)...", cc.config.Workers)
	// Start workers
	for i := 0; i < cc.config.Workers; i++ {
		wg.Add(1)
		go cc.startWorker(i, ctx, &wg)
	}

	log.Printf("Sending %d cards to workers...", len(cc.cards))
	// Send cards to the channel
	for _, card := range cc.cards {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, closing cardsChan and waiting for workers...")
			close(cc.cardsChan)
			wg.Wait()
			log.Println("All workers finished.")
			return
		case cc.cardsChan <- card:
			log.Printf("Card '%s' sent to worker.", card.GetFullName())
		}
	}
	log.Println("All cards have been sent to workers. Closing cardsChan.")
	close(cc.cardsChan)
	wg.Wait()
	close(cc.outputChan)
	log.Println("All workers have finished their work.")
}

func (cc *CardConjurer) startWorker(id int, ctx context.Context, wg *sync.WaitGroup) {
	log.Printf("Starting worker %d", id)

	defer func() {
		log.Printf("Worker %d: finished", id)
		wg.Done()
	}()

	w := newWorker(id, cc.config)
	w.startWorker(ctx, cc.cardsChan, cc.outputChan)
}
