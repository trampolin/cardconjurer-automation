package decklist_parser

import (
	"cardconjurer-automation/pkg/common"
	"encoding/csv"
	"os"
	"strconv"
)

type DecklistParser struct {
	filename string
	decklist []common.CardInfo
}

func New(filename string) (*DecklistParser, error) {
	csvParser := &DecklistParser{
		filename: filename,
	}

	// check if the file exists:
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, err
	}

	return csvParser, nil
}

func (c *DecklistParser) Parse() ([]common.CardInfo, error) {
	// Open the CSV file
	file, err := os.Open(c.filename)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	// Read the CSV file
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 4
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		count, err := strconv.Atoi(record[0])
		if err != nil {
			continue
		}
		name := record[1]
		set := record[2]
		collectorNumber := record[3]

		card := &Card{
			Count:           count,
			Name:            name,
			Set:             set,
			CollectorNumber: collectorNumber,
		}

		c.decklist = append(c.decklist, card)
	}

	return c.decklist, nil
}
