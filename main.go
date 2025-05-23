package main

import (
	"cardconjurer-automation/pkg/cardconjurer"
	"cardconjurer-automation/pkg/common"
	"cardconjurer-automation/pkg/decklist_parser"
	"cardconjurer-automation/pkg/mpc"
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	// Flags definieren (csvFile ist kein Flag mehr)
	baseUrl := flag.String("base-url", "", "base url")
	output := flag.String("output", "", "Pfad zum Ausgabeverzeichnis der Karten")
	input := flag.String("input", "", "Pfad zum Artwork-Verzeichnis")
	cardsFilter := flag.String("cards-filter", "", "Kartenfilter (optional, kommasepariert)")
	workers := flag.Int("workers", 2, "Anzahl der Worker")
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

	// Projektname aus csvFile ableiten: Dateiname, lowercase, snake_case, keine Sonderzeichen, ohne .csv
	projectName := strings.TrimSuffix(filepath.Base(csvFile), filepath.Ext(csvFile))
	projectName = strings.ToLower(projectName)
	projectName = strings.ReplaceAll(projectName, " ", "_")
	// Entferne alle Zeichen außer Buchstaben, Zahlen und Unterstrich
	var sanitized strings.Builder
	for _, r := range projectName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			sanitized.WriteRune(r)
		}
	}
	projectName = sanitized.String()

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

	var cardList []common.CardInfo
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
		cardList = append(cardList, card)
	}

	wg := &sync.WaitGroup{}
	ctx := context.Background()

	ccCfg := &cardconjurer.Config{
		Workers:            *workers,
		BaseUrl:            *baseUrl,
		InputArtworkFolder: *input,
		OutputCardsFolder:  *output,
		ProjectName:        projectName,
	}

	cc, err := cardconjurer.New(ccCfg, cardList)
	if err != nil {
		log.Fatal(err)
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		cc.Run(ctx)
	}()

	mpcCfg := &mpc.Config{
		ProjectPath: filepath.Dir(csvFile),
		ProjectName: projectName,
	}

	mpc := mpc.New(mpcCfg)

	go func() {
		defer wg.Done()
		mpc.Run(cc.GetOutputChan(), ctx)
	}()

	wg.Wait()
}
