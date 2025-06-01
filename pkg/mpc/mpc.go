package mpc

import (
	"cardconjurer-automation/pkg/common"
	"context"
	"fmt"
	"go.uber.org/zap"
	"os"
)

type Config struct {
	ProjectPath string
	ProjectName string
}

type MPC struct {
	config *Config
	logger *zap.SugaredLogger
}

func New(config *Config, logger *zap.SugaredLogger) *MPC {
	return &MPC{
		config: config,
		logger: logger,
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
				m.logger.Info("Card channel closed")
				return nil
			}

			m.logger.Infow("Adding card to xml", "card", card.GetFullName())
			order.AddFront(card)
			xml, err := order.GetXml()
			if err != nil {
				m.logger.Errorf("Error generating XML: %v", err)
				continue
			}

			// Save XML to file
			filePath := fmt.Sprintf("%s/%s.xml", m.config.ProjectPath, m.config.ProjectName)
			err = os.WriteFile(filePath, xml, 0644)
			if err != nil {
				m.logger.Errorf("Error writing XML file: %v", err)
			}
		}
	}
}
