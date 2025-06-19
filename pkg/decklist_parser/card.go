package decklist_parser

import (
	"fmt"
	"strings"
)

type Card struct {
	Count           int
	Name            string
	NameFront       string
	NameBack        string
	Set             string
	CollectorNumber string
}

func (c *Card) String() string {
	return c.GetFullName()
}

func (c *Card) GetFullName() string {
	return fmt.Sprintf("%s (%s #%s)", c.Name, strings.ToUpper(c.Set), c.CollectorNumber)
}

func (c *Card) getSanitizedName(name string) string {
	// Everything in lowercase
	name = strings.ToLower(name)
	// Replace spaces with underscores
	name = strings.ReplaceAll(name, " ", "_")
	// Replace typographic apostrophe with straight apostrophe
	name = strings.ReplaceAll(name, "’", "'")
	// Remove all characters except letters, numbers, or underscores
	var sanitized strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			sanitized.WriteRune(r)
		}
	}
	return sanitized.String()
}

func (c *Card) GetSanitizedNameFront() string {
	return c.getSanitizedName(c.NameFront)
}

func (c *Card) GetSanitizedNameBack() string {
	if c.NameBack == "" {
		return ""
	}
	return c.getSanitizedName(c.NameBack)
}

func (c *Card) GetCount() int {
	return c.Count
}

func (c *Card) GetNameFront() string {
	return c.NameFront
}

func (c *Card) GetNameBack() string {
	return c.NameBack
}

func (c *Card) GetSet() string {
	return c.Set
}

func (c *Card) GetCollectorNumber() string {
	return c.CollectorNumber
}
