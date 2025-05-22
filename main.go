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
	// Flags definieren
	baseUrl := flag.String("base-url", "", "base url")
	csvFile := flag.String("csv-file", "", "Pfad zur CSV-Datei")
	output := flag.String("output", "output", "Pfad zum Ausgabeverzeichnis")
	cards := flag.String("cards", "", "Kartenfilter (optional)")
	skipImages := flag.Bool("skip-images", false, "Bilder überspringen")
	flag.Parse()

	dp, err := decklist_parser.New(*csvFile)
	if err != nil {
		log.Fatal(err)
	}

	decklist, err := dp.Parse()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("card filter: %s", *cards)

	var cardList []cardconjurer.CardInfo
	for _, card := range decklist {
		if *cards != "" {
			if *cards != card.GetName() {
				continue
			}
		}
		cardList = append(cardList, &card)
	}

	cfg := &cardconjurer.Config{
		Workers: 1,
		BaseUrl: *baseUrl,
	}
	cc, err := cardconjurer.New(cfg, cardList)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	cc.Run(ctx)

	//for _, card := range decklist {
	//	log.Println(card)
	//}

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
	fmt.Println("csv-file:", *csvFile)
	fmt.Println("output:", *output)
	fmt.Println("cards:", *cards)
	fmt.Println("skip-images:", *skipImages)
}
