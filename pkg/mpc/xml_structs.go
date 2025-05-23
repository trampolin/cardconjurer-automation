package mpc

type OrderDetails struct {
	Quantity int    `xml:"quantity"`
	Bracket  int    `xml:"bracket"`
	Stock    string `xml:"stock"`
	Foil     bool   `xml:"foil"`
}
