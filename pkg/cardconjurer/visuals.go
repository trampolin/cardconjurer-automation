package cardconjurer

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"os"
	"time"
)

func (cc *CardConjurer) addMargin(browserCtx context.Context) error {
	// Klick auf das Frame-Tab und warte, bis das Dropdown sichtbar ist
	log.Println("Starte Margin-Import")
	if err := chromedp.Run(browserCtx,
		chromedp.Click(`h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="frame"]`),
		chromedp.WaitVisible(`#selectFrameGroup`, chromedp.ByID),
	); err != nil {
		return err
	}

	// Wähle "Margin" im Dropdown und warte, bis der Button bereit ist
	log.Println("Wähle 'Margin' im Frame-Dropdown")
	if err := chromedp.Run(browserCtx,
		chromedp.SetValue(`#selectFrameGroup`, "Margin"),
		chromedp.WaitReady(`#addToFull`, chromedp.ByID),
	); err != nil {
		return err
	}

	// Warte, bis das gewünschte Bild-Element geladen ist
	log.Println("Warte auf das Margin-Bild-Element...")
	if err := chromedp.Run(browserCtx,
		chromedp.WaitReady(`img[src="/img/frames/margins/blackBorderExtensionThumb.png"]`),
	); err != nil {
		return err
	}

	log.Println("Klicke auf 'addToFull'-Button...")
	if err := chromedp.Run(browserCtx,
		chromedp.Click(`#addToFull`),
	); err != nil {
		return err
	}

	// Nach dem Klick auf 'addToFull': Warte, bis sich das Canvas-Element (#previewCanvas) ändert.
	// Da sich das Canvas nicht direkt vergleichen lässt, kann man z.B. die Größe, ein Attribut oder einen Hash des Bildinhalts beobachten.
	// Hier: Wir lesen vor dem Klick ein DataURL-Snapshot und warten, bis sich dieser ändert.

	log.Println("Warte auf Änderung des Canvas nach 'addToFull'...")
	var oldDataURL string
	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(`document.getElementById('previewCanvas')?.toDataURL()`, &oldDataURL),
	); err != nil {
		log.Printf("Konnte Canvas-DataURL nicht lesen: %v", err)
		// kein fataler Fehler, weitermachen
	}

	if oldDataURL != "" {
		ctx, cancel := context.WithTimeout(browserCtx, 10*time.Second)
		defer cancel()
		var newDataURL string
		err := chromedp.Run(ctx,
			chromedp.Poll(`(() => {
				const c = document.getElementById('previewCanvas');
				return c && c.toDataURL() !== "`+oldDataURL+`";
			})()`, nil, chromedp.WithPollingInterval(200*time.Millisecond)),
			chromedp.Evaluate(`document.getElementById('previewCanvas')?.toDataURL()`, &newDataURL),
		)
		if err != nil {
			log.Printf("Timeout oder Fehler beim Warten auf Canvas-Update: %v", err)
		} else {
			log.Printf("Canvas wurde aktualisiert.")
		}
	} else {
		log.Println("Kein Canvas-DataURL gefunden, kann nicht auf Änderung warten.")
	}

	log.Println("Margin-Import abgeschlossen")
	return nil
}

func (cc *CardConjurer) replaceArtwork(card CardInfo, browserCtx context.Context) error {
	if cc.config.InputArtworkFolder == "" {
		//return nil
	}

	// Klick auf das Artwork-Tab und warte, bis das File-Input sichtbar ist
	log.Println("Starte Artwork-Import")
	inputSelector := `input[type="file"][accept*=".png"][data-dropfunction="uploadArt"]`
	err := cc.openTab(
		browserCtx,
		"art",
		inputSelector,
	)
	if err != nil {
		return err
	}

	// Prüfe, ob eine passende PNG-Datei im Artwork-Ordner existiert
	filename := fmt.Sprintf("%s.png", card.GetName())
	filepath := fmt.Sprintf("%s/%s", cc.config.InputArtworkFolder, filename)
	if _, err := os.Stat(filepath); err != nil {
		if os.IsNotExist(err) {
			log.Printf("Artwork-Datei nicht gefunden: %s", filepath)
			return nil
		}

		return err
	}

	log.Printf("Artwork-Datei gefunden: %s", filepath)
	// Setze den Dateipfad als Wert für das File-Input
	if err := chromedp.Run(browserCtx,
		chromedp.SetUploadFiles(inputSelector, []string{filepath}),
	); err != nil {
		log.Printf("Fehler beim Setzen der Artwork-Datei: %v", err)
		return err
	}
	log.Printf("Artwork-Datei %s erfolgreich gesetzt.", filepath)

	return nil
}

func (cc *CardConjurer) removeSetSymbol(browserCtx context.Context) error {
	//buttonSelector := `button.input.margin-bottom[onclick*="removeSetSymbol()"]`
	buttonSelector := `#creator-menu-setSymbol > div:nth-child(3) > button`

	err := cc.openTab(browserCtx, "setSymbol", buttonSelector)
	if err != nil {
		return err
	}

	// Klicke auf den Button, um das Set-Symbol zu entfernen
	if err := chromedp.Run(browserCtx,
		chromedp.Click(buttonSelector),
		chromedp.Sleep(250*time.Millisecond),
	); err != nil {
		return err
	}

	return nil
}
