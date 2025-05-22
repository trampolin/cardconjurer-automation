package main

import (
	"cardconjurer-automation/pkg/cardconjurer"
	"cardconjurer-automation/pkg/decklist_parser"
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Flags definieren (csvFile ist kein Flag mehr)
	baseUrl := flag.String("base-url", "", "base url")
	output := flag.String("output", "", "Pfad zum Ausgabeverzeichnis der Karten")
	input := flag.String("input", "", "Pfad zum Artwork-Verzeichnis")
	cardsFilter := flag.String("cards-filter", "", "Kartenfilter (optional, kommasepariert)")
	//skipImages := flag.Bool("skip-images", false, "Bilder überspringen")
	flag.Parse()

	// csvFile als Pflicht-Parameter (erstes Argument nach den Flags)
	if flag.NArg() < 1 {
		log.Println("Fehler: Pfad zur CSV-Datei muss als Argument übergeben werden.")
		log.Println("Aufruf: ./programm [flags] <csv-file>")
		flag.Usage()
		os.Exit(1)
	}
	csvFile := flag.Arg(0)

	// Wenn input leer ist, setze auf <csvFile>/artworks
	if *input == "" && csvFile != "" {
		*input = filepath.Join(filepath.Dir(csvFile), "artworks")
	}

	// Wenn output leer ist, setze auf <csvFile>/cards
	if *output == "" && csvFile != "" {
		*output = filepath.Join(filepath.Dir(csvFile), "cards")
	}

	// Erstelle Artwork- und Cards-Ordner, falls sie nicht existieren
	if err := os.MkdirAll(*input, 0755); err != nil {
		log.Fatalf("Konnte Artwork-Ordner nicht erstellen: %v", err)
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		log.Fatalf("Konnte Cards-Ordner nicht erstellen: %v", err)
	}

	dp, err := decklist_parser.New(csvFile)
	if err != nil {
		log.Fatal(err)
	}

	decklist, err := dp.Parse()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("card filter: %s", *cardsFilter)

	var cardList []cardconjurer.CardInfo
	var filterSet map[string]struct{}
	if *cardsFilter != "" {
		filterSet = make(map[string]struct{})
		for _, name := range strings.Split(*cardsFilter, ",") {
			filterSet[strings.TrimSpace(name)] = struct{}{}
		}
	}
	for _, card := range decklist {
		if filterSet != nil {
			if _, ok := filterSet[card.GetName()]; !ok {
				continue
			}
		}
		cardList = append(cardList, &card)
	}

	cfg := &cardconjurer.Config{
		Workers:            1,
		BaseUrl:            *baseUrl,
		InputArtworkFolder: *input,
		OutputCardsFolder:  *output,
	}
	cc, err := cardconjurer.New(cfg, cardList)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	cc.Run(ctx)
}
