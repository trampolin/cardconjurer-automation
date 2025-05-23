package mpc

import (
	"cardconjurer-automation/pkg/common"
	"encoding/xml"
)

// Brackets für Kartenmengen
var brackets = []int{18, 36, 55, 72, 90, 108, 126, 144, 162, 180, 198, 216, 234, 396, 504, 612}

type Order struct {
	Details  *OrderDetails `xml:"details"`
	Fronts   *XmlCards     `xml:"fronts"`
	Backs    *XmlCards     `xml:"backs"`
	CardBack string        `xml:"cardback"`
}

func NewOrder() *Order {
	return &Order{
		Details: &OrderDetails{
			Quantity: 0,
			Bracket:  18,
			Stock:    "",
			Foil:     false,
		},
		Fronts: &XmlCards{},
		Backs:  &XmlCards{},
	}
}

func (o *Order) AddFront(card common.CardInfo) {
	o.Details.Quantity += card.GetCount()
	o.Fronts.AddCard(card)
	o.UpdateBracket()
}

func (o *Order) UpdateBracket() {
	qty := o.Details.Quantity
	for _, bracket := range brackets {
		if qty <= bracket {
			o.Details.Bracket = bracket
			return
		}
	}

	o.Details.Bracket = brackets[len(brackets)-1]
}

func (o *Order) GetXml() ([]byte, error) {
	return xml.MarshalIndent(o, "", "  ")
}
