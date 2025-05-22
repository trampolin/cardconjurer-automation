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

	// Worker starten
	for i := 0; i < cc.config.Workers; i++ {
		go cc.startWorker(i, ctx, &wg)
	}

	// Karten in den Channel senden
	for _, card := range cc.cards {
		select {
		case <-ctx.Done():
			close(cc.cardsChan)
			log.Println("Context done, closing cards channel")
			wg.Wait()
			log.Println("All workers finished")
			return
		case cc.cardsChan <- card:
		}
	}
	log.Println("All cards sent to workers")
	close(cc.cardsChan)
	wg.Wait()
	log.Println("All workers finished")
	return
}

func (cc *CardConjurer) startWorker(id int, ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	browserCtx, err := cc.openBrowser(ctx)
	if err != nil {
		log.Printf("Worker %d: error opening browser: %v", id, err)
		return
	}

	defer func() {
		cc.closeBrowser(browserCtx)
		log.Printf("Worker %d: browser closed", id)
	}()

	log.Printf("Starting worker %d", id)
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d: context done, exiting", id)
			return
		case card, ok := <-cc.cardsChan:
			if !ok {
				log.Printf("Worker %d: cards channel closed", id)
				return
			}

			log.Printf("Worker %d starting card: %s", id, card.GetFullName())

			err := cc.importCard(card, browserCtx)
			if err != nil {
				log.Println(err)
				continue
			}

			err = cc.addMargin(browserCtx)
			if err != nil {
				log.Println(err)
				continue
			}

			time.Sleep(time.Millisecond * 5000)
		}
	}
}
