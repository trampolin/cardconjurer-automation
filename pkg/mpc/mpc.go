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

	order := NewOrder(m.config.ProjectName)

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
			filePath := fmt.Sprintf("%s/%s.xml", m.config.ProjectPath, m.config.ProjectName)
			err = os.WriteFile(filePath, xml, 0644)

		}
	}
}
