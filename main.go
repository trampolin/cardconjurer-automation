package main

import (
	"cardconjurer-automation/pkg/cardconjurer"
	"cardconjurer-automation/pkg/common"
	"cardconjurer-automation/pkg/decklist_parser"
	"cardconjurer-automation/pkg/mpc"
	"context"
	"flag"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	baseUrl := flag.String("base-url", "", "base url")
	output := flag.String("output", "", "Pfad zum Ausgabeverzeichnis der Karten")
	input := flag.String("input", "", "Pfad zum Artwork-Verzeichnis")
	cardsFilter := flag.String("cards-filter", "", "Kartenfilter (optional, kommasepariert)")
	workers := flag.Int("workers", 2, "Anzahl der Worker")
	flag.Parse()

	if flag.NArg() < 1 {
		sugar.Error("Fehler: Pfad zur CSV-Datei muss als Argument übergeben werden.")
		sugar.Info("Aufruf: ./programm [flags] <csv-file>")
		flag.Usage()
		os.Exit(1)
	}
	csvFile := flag.Arg(0)

	projectName := strings.TrimSuffix(filepath.Base(csvFile), filepath.Ext(csvFile))
	projectName = strings.ToLower(projectName)
	projectName = strings.ReplaceAll(projectName, " ", "_")
	var sanitized strings.Builder
	for _, r := range projectName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			sanitized.WriteRune(r)
		}
	}
	projectName = sanitized.String()

	if *input == "" && csvFile != "" {
		*input = filepath.Join(filepath.Dir(csvFile), "artworks")
	}

	if *output == "" && csvFile != "" {
		*output = filepath.Join(filepath.Dir(csvFile), "cards")
	}

	if err := os.MkdirAll(*input, 0755); err != nil {
		sugar.Fatalf("Konnte Artwork-Ordner nicht erstellen: %v", err)
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		sugar.Fatalf("Konnte Cards-Ordner nicht erstellen: %v", err)
	}

	dp, err := decklist_parser.New(csvFile)
	if err != nil {
		sugar.Fatal(err)
	}

	decklist, err := dp.Parse()
	if err != nil {
		sugar.Fatal(err)
	}

	sugar.Infof("card filter: %s", *cardsFilter)

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
		sugar.Fatal(err)
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
