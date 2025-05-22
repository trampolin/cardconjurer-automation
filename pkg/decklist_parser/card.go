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

func (c *Card) GetSanitizedName() string {
	// Alles in Kleinbuchstaben
	name := strings.ToLower(c.Name)
	// Ersetze Leerzeichen durch Unterstrich
	name = strings.ReplaceAll(name, " ", "_")
	// Ersetze typografisches Apostroph durch normales
	name = strings.ReplaceAll(name, "’", "'")
	// Entferne alle Zeichen, die keine Buchstaben, Zahlen oder Unterstriche sind
	var sanitized strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			sanitized.WriteRune(r)
		}
	}
	return sanitized.String()
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
