package cardconjurer

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"os"
	"strings"
	"time"
)

func (cc *CardConjurer) openBrowser(parentCtx context.Context) (context.Context, error) {
	log.Println("Starte neuen Browser...")
	dir, err := os.MkdirTemp("", "chromedp-example")
	if err != nil {
		log.Printf("Fehler beim Erstellen des Temp-Verzeichnisses: %v", err)
		return nil, err
	}
	// os.RemoveAll(dir) sollte vom Aufrufer übernommen werden

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.UserDataDir(dir),
		chromedp.Flag("headless", false),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(parentCtx, opts...)
	// cancel sollte vom Aufrufer übernommen werden

	taskCtx, cancel2 := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	// cancel2 sollte vom Aufrufer übernommen werden

	if err := chromedp.Run(taskCtx); err != nil {
		cancel2()
		cancel()
		return nil, err
	}

	log.Printf("Opening browser at %s", cc.config.BaseUrl)
	if err := chromedp.Run(taskCtx,
		chromedp.Navigate(cc.config.BaseUrl),
		// Warte, bis das Dokument vollständig geladen ist
		chromedp.WaitReady("body"),
	); err != nil {
		cancel2()
		cancel()
		return nil, err
	}

	return taskCtx, nil
}

func (cc *CardConjurer) closeBrowser(browserCtx context.Context) {
	log.Println("Schließe Browser...")
	if err := chromedp.Cancel(browserCtx); err != nil {
		log.Printf("Error closing browser: %v", err)
	}
	log.Println("Browser closed")
}

func (cc *CardConjurer) importCard(cardData CardInfo, browserCtx context.Context) error {
	log.Printf("Starte Import für Karte: %s", cardData.GetFullName())

	err := cc.openTab(browserCtx, "import", "#import-name")
	if err != nil {
		return err
	}

	// Klick auf das Import-Tab und warte, bis das Dropdown sichtbar ist
	if err := chromedp.Run(browserCtx,
		chromedp.Click(`h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="import"]`),
		chromedp.WaitVisible(`#autoFrame`, chromedp.ByID),
	); err != nil {
		log.Printf("Fehler beim Öffnen des Import-Tabs: %v", err)
		return err
	}

	log.Println("Import-Tab geöffnet, wähle Option 'Seventh' im Dropdown.")
	// Wähle Option 'Seventh' im Dropdown mit id 'autoFrame' und warte, bis die Checkbox bereit ist
	if err := chromedp.Run(browserCtx,
		chromedp.SetValue(`#autoFrame`, "Seventh"),
		chromedp.WaitReady(`#importAllPrints`, chromedp.ByID),
	); err != nil {
		return err
	}

	// Weitere Aktionen: check_import_all_prints und load_card
	log.Println("Überprüfe Checkbox 'Import All Prints' und lade Karte...")
	if err := cc.checkImportAllPrints(browserCtx); err != nil {
		return err
	}
	log.Println("Lade Karte...")
	if err := cc.loadCard(cardData, browserCtx); err != nil {
		return err
	}

	return nil
}

func (cc *CardConjurer) checkImportAllPrints(browserCtx context.Context) error {
	var checked bool
	// Prüfe, ob die Checkbox gecheckt ist
	err := chromedp.Run(browserCtx,
		chromedp.EvaluateAsDevTools(`document.querySelector('#importAllPrints')?.checked`, &checked),
	)
	if err != nil {
		return err
	}
	if !checked {
		log.Println("Checkbox 'Import All Prints' ist nicht gecheckt, klicke darauf.")
		// Klicke auf das Parent-Element der Checkbox und warte, bis die Checkbox wieder sichtbar ist
		err = chromedp.Run(browserCtx,
			chromedp.EvaluateAsDevTools(`document.querySelector('#importAllPrints').parentElement.click()`, nil),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cc *CardConjurer) loadCard(cardData CardInfo, browserCtx context.Context) error {
	// Vor dem Eintragen des Namens: Alle Optionen aus dem Dropdown entfernen
	log.Println("Lösche alle Optionen aus #import-index vor neuer Suche...")
	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(`document.querySelectorAll('#import-index option').forEach(o => o.remove())`, nil),
	); err != nil {
		log.Printf("Konnte Optionen im Dropdown nicht löschen: %v", err)
		// kein fataler Fehler, weitermachen
	}

	// Tab drücken: Fokus setzen, dann Tab-Key als RawEvent senden, danach warten bis das Dropdown bereit ist
	if err := chromedp.Run(browserCtx,
		chromedp.WaitVisible(`#import-name`, chromedp.ByID),
		chromedp.WaitReady(`#import-name`, chromedp.ByID),
		chromedp.SetValue(`#import-name`, cardData.GetName(), chromedp.ByID),
		chromedp.Focus(`#import-name`, chromedp.ByID),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.SendKeys(`#import-name`, "\t", chromedp.ByID).Do(ctx)
		}),
		// Warte, bis mindestens eine Option im Dropdown geladen ist
		chromedp.Poll(`document.querySelectorAll('#import-index option').length > 1`, nil, chromedp.WithPollingInterval(100*time.Millisecond)),
		chromedp.WaitReady(`#import-index`, chromedp.ByID),
	); err != nil {
		return err
	}

	// Card-Version-String bauen
	cardVersion := cardData.GetFullName()

	// Alle Optionen im Dropdown abfragen
	var optionTexts []string
	var optionValues []string
	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(`Array.from(document.querySelectorAll('#import-index option')).map(o => o.textContent.trim())`, &optionTexts),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('#import-index option')).map(o => o.value)`, &optionValues),
	); err != nil {
		return err
	}

	// Passende Option suchen (auch auf exakte Übereinstimmung nach Trim)
	var valueToSelect string
	for i, text := range optionTexts {
		if text == cardVersion {
			valueToSelect = optionValues[i]
			break
		}
	}
	if valueToSelect == "" {
		// Versuche es mit Whitespace-ignorierender Suche
		for i, text := range optionTexts {
			if strings.TrimSpace(text) == strings.TrimSpace(cardVersion) {
				valueToSelect = optionValues[i]
				break
			}
		}
	}

	if valueToSelect != "" {
		// Option auswählen und warten, bis das Dropdown wieder bereit ist
		if err := chromedp.Run(browserCtx,
			chromedp.SetAttributeValue(fmt.Sprintf(`#import-index option[value="%s"]`, valueToSelect), "selected", "true"),
			chromedp.SetValue(`#import-index`, valueToSelect),
			chromedp.WaitReady(`#import-index`, chromedp.ByID),
		); err != nil {
			return err
		}
	} else {
		log.Printf("Warnung: Keine passende Karten-Version gefunden: %s", cardVersion)
	}

	return nil
}

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
	if cc.config.ArtworkFolder == "" {
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
	filepath := fmt.Sprintf("%s/%s", cc.config.ArtworkFolder, filename)
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
	); err != nil {
		return err
	}

	return nil
}

// openTab öffnet ein Tab anhand des Tab-Namens (z.B. "import", "frame").
// Es kann auf beliebig viele Selektoren nach dem Klick gewartet werden.
func (cc *CardConjurer) openTab(ctx context.Context, tabName string, waitForSelectors ...string) error {
	selector := fmt.Sprintf(`h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="%s"]`, tabName)
	log.Printf("Öffne Tab: %s", tabName)
	actions := []chromedp.Action{
		chromedp.Click(selector),
	}
	for _, sel := range waitForSelectors {
		if sel != "" {
			log.Printf("Warte auf Element nach Tab-Wechsel: %s", sel)
			actions = append(actions, chromedp.WaitVisible(sel))
		}
	}
	return chromedp.Run(ctx, actions...)
}
