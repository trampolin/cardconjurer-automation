package decklist_parser

import (
	"fmt"
	"strings"
)

type Card struct {
	Count           int
	Name            string
	Set             string
	CollectorNumber string
}

func (c *Card) String() string {
	return c.GetName()
}

func (c *Card) GetFullName() string {
	return fmt.Sprintf("%s (%s #%s)", c.Name, strings.ToUpper(c.Set), c.CollectorNumber)
}

func (c *Card) GetCount() int {
	return c.Count
}

func (c *Card) GetName() string {
	return c.Name
}

func (c *Card) GetSet() string {
	return c.Set
}

func (c *Card) GetCollectorNumber() string {
	return c.CollectorNumber
}
