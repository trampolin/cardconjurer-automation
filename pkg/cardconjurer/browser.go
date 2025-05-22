package cardconjurer

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"os"
	"time"
)

func (cc *CardConjurer) openBrowser(parentCtx context.Context) (context.Context, error) {
	dir, err := os.MkdirTemp("", "chromedp-example")
	if err != nil {
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
	); err != nil {
		cancel2()
		cancel()
		return nil, err
	}

	return taskCtx, nil
}

func (cc *CardConjurer) closeBrowser(browserCtx context.Context) {
	if err := chromedp.Cancel(browserCtx); err != nil {
		log.Printf("Error closing browser: %v", err)
	}
	log.Println("Browser closed")
}

func (cc *CardConjurer) importCard(cardData CardInfo, browserCtx context.Context) error {
	// Klick auf das Import-Tab
	if err := chromedp.Run(browserCtx,
		chromedp.Click(`h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="import"]`),
	); err != nil {
		return err
	}

	time.Sleep(300 * time.Millisecond)

	// Wähle Option 'Seventh' im Dropdown mit id 'autoFrame'
	if err := chromedp.Run(browserCtx,
		chromedp.SetValue(`#autoFrame`, "Seventh"),
	); err != nil {
		return err
	}

	time.Sleep(500 * time.Millisecond)

	// Weitere Aktionen: check_import_all_prints und load_card
	if err := cc.checkImportAllPrints(browserCtx); err != nil {
		return err
	}
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
		// Klicke auf das Parent-Element der Checkbox
		err = chromedp.Run(browserCtx,
			chromedp.EvaluateAsDevTools(`document.querySelector('#importAllPrints').parentElement.click()`, nil),
		)
		if err != nil {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

func (cc *CardConjurer) loadCard(cardData CardInfo, browserCtx context.Context) error {
	// Name ins Importfeld eintragen
	if err := chromedp.Run(browserCtx,
		chromedp.SetValue(`#import-name`, cardData.GetName(), chromedp.ByID),
	); err != nil {
		return err
	}

	time.Sleep(200 * time.Millisecond)

	// Tab drücken: Fokus setzen, dann Tab-Key als RawEvent senden
	if err := chromedp.Run(browserCtx,
		chromedp.Focus(`#import-name`, chromedp.ByID),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.SendKeys(`#import-name`, "\t", chromedp.ByID).Do(ctx)
		}),
	); err != nil {
		return err
	}

	time.Sleep(1000 * time.Millisecond)

	// Card-Version-String bauen
	cardVersion := cardData.GetFullName()

	// Alle Optionen im Dropdown abfragen
	var optionTexts []string
	var optionValues []string
	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(`Array.from(document.querySelectorAll('#import-index option')).map(o => o.textContent)`, &optionTexts),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('#import-index option')).map(o => o.value)`, &optionValues),
	); err != nil {
		return err
	}

	// Passende Option suchen
	var valueToSelect string
	for i, text := range optionTexts {
		if text == cardVersion {
			valueToSelect = optionValues[i]
			break
		}
	}

	if valueToSelect != "" {
		// Option auswählen
		if err := chromedp.Run(browserCtx,
			chromedp.SetAttributeValue(fmt.Sprintf(`#import-index option[value="%s"]`, valueToSelect), "selected", "true"),
			chromedp.SetValue(`#import-index`, valueToSelect),
		); err != nil {
			return err
		}
		time.Sleep(2000 * time.Millisecond)
	} else {
		log.Printf("Warnung: Keine passende Karten-Version gefunden: %s", cardVersion)
	}

	return nil
}

func (cc *CardConjurer) addMargin(browserCtx context.Context) error {
	// Klick auf das Frame-Tab
	if err := chromedp.Run(browserCtx,
		chromedp.Click(`h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="frame"]`),
	); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)

	// Wähle "Margin" im Dropdown mit id 'selectFrameGroup'
	if err := chromedp.Run(browserCtx,
		chromedp.SetValue(`#selectFrameGroup`, "Margin"),
	); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)

	// Klicke auf den Button mit id 'addToFull'
	if err := chromedp.Run(browserCtx,
		chromedp.Click(`#addToFull`),
	); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)

	return nil
}
