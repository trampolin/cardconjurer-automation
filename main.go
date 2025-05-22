package main

import (
	"bytes"
	"cardconjurer-automation/pkg/cardconjurer"
	"cardconjurer-automation/pkg/decklist_parser"
	"context"
	"flag"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// Flags definieren (csvFile ist kein Flag mehr)
	baseUrl := flag.String("base-url", "", "base url")
	output := flag.String("output", "", "Pfad zum Ausgabeverzeichnis der Karten")
	input := flag.String("input", "", "Pfad zum Artwork-Verzeichnis")
	cardsFilter := flag.String("cards-filter", "", "Kartenfilter (optional)")
	skipImages := flag.Bool("skip-images", false, "Bilder überspringen")
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
	for _, card := range decklist {
		if *cardsFilter != "" {
			if *cardsFilter != card.GetName() {
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

	return

	dir, err := os.MkdirTemp("", "chromedp-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.UserDataDir(dir),
		// Headless deaktivieren, damit der Browser sichtbar ist
		chromedp.Flag("headless", false),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// also set up a custom logger
	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// ensure that the browser process is started
	if err := chromedp.Run(taskCtx); err != nil {
		log.Fatal(err)
	}

	// Browser öffnen und google.com laden
	if err := chromedp.Run(taskCtx,
		chromedp.Navigate(*baseUrl),
	); err != nil {
		log.Fatal(err)
	}

	//// Beispiel: Auf einen Button klicken (z.B. "Ich stimme zu" auf Google)
	//if err := chromedp.Run(taskCtx,
	//	// Passe den Selektor ggf. an die Seite an!
	//	chromedp.Click(`#L2AGLb`, chromedp.NodeVisible),
	//); err != nil {
	//	log.Println("Klicken fehlgeschlagen:", err)
	//}

	// Optional: Warten, damit das Browserfenster offen bleibt
	fmt.Println("Drücke Enter zum Beenden...")
	fmt.Scanln()

	path := filepath.Join(dir, "DevToolsActivePort")
	bs, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	lines := bytes.Split(bs, []byte("\n"))
	fmt.Printf("DevToolsActivePort has %d lines\n", len(lines))

	// Beispiel: Ausgabe der Flag-Werte
	fmt.Println("csv-file:", csvFile)
	fmt.Println("output:", *output)
	fmt.Println("cardsFilter:", *cardsFilter)
	fmt.Println("skip-images:", *skipImages)
}
