package mpc

type XmlCard struct {
	ID    string `xml:"id"`
	Slots string `xml:"slots"`
	Name  string `xml:"name"`
	Query string `xml:"query"`
}
