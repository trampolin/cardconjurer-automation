package mpc

import (
	"cardconjurer-automation/pkg/common"
	"fmt"
	"strconv"
)

type XmlCards struct {
	projectName string
	Cards       []XmlCard `xml:"card"`
}

func (xc *XmlCards) AddCard(card common.CardInfo) {

	for i := 0; i < card.GetCount(); i++ {
		xc.Cards = append(xc.Cards, XmlCard{
			ID:    card.GetSanitizedName(),
			Slots: strconv.Itoa(len(xc.Cards)),
			Name:  fmt.Sprintf("%s_%s.png", xc.projectName, card.GetSanitizedName()),
			Query: card.GetName(),
		})
	}
}
