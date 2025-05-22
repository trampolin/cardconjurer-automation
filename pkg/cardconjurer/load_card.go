package cardconjurer

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"strings"
	"time"
)

func (w *worker) importCard(cardData CardInfo, browserCtx context.Context) error {
	log.Printf("Starte Import für Karte: %s", cardData.GetFullName())

	err := w.openTab(browserCtx, "import", "#import-name")
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
	if err := w.checkImportAllPrints(browserCtx); err != nil {
		return err
	}
	log.Println("Lade Karte...")
	if err := w.loadCard(cardData, browserCtx); err != nil {
		return err
	}

	return nil
}

func (w *worker) checkImportAllPrints(browserCtx context.Context) error {
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

func (w *worker) loadCard(cardData CardInfo, browserCtx context.Context) error {
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
