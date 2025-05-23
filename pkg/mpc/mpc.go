package mpc

import (
	"cardconjurer-automation/pkg/common"
	"context"
	"fmt"
	"log"
	"os"
)

type Config struct {
	ProjectPath string
	ProjectName string
}

type MPC struct {
	config *Config
}

func New(config *Config) *MPC {
	return &MPC{
		config: config,
	}
}

func (m *MPC) Run(cards <-chan common.CardInfo, ctx context.Context) error {

	order := NewOrder()

	for {
		select {
		case <-ctx.Done():
			return nil
		case card, ok := <-cards:
			if !ok {
				log.Println("Card channel closed")
				return nil
			}

			order.AddFront(card)
			xml, err := order.GetXml()
			if err != nil {
				log.Printf("Error generating XML: %v", err)
				continue
			}

			// Save XML to file
			fileName := fmt.Sprintf("%s_%s.xml", m.config.ProjectName, card.GetSanitizedName())
			filePath := fmt.Sprintf("%s/%s", m.config.ProjectPath, fileName)
			err = os.WriteFile(filePath, xml, 0644)

		}
	}
}
