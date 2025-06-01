package cardconjurer

import (
	"cardconjurer-automation/pkg/common"
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"strings"
	"time"
)

func (w *worker) importCard(cardData common.CardInfo, browserCtx context.Context) error {
	w.logger.Infof("Starting import for card: %s", cardData.GetFullName())

	err := w.openTab(browserCtx, "import", "#import-name")
	if err != nil {
		return err
	}

	// Click import tab and wait for dropdown to be visible
	if err := chromedp.Run(browserCtx,
		chromedp.Click(`h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="import"]`),
		chromedp.WaitVisible(`#autoFrame`, chromedp.ByID),
	); err != nil {
		w.logger.Errorf("Error opening import tab: %v", err)
		return err
	}

	w.logger.Info("Import tab opened, selecting option 'Seventh' in dropdown.")
	// Select option 'Seventh' in dropdown with id 'autoFrame' and wait for checkbox to be ready
	if err := chromedp.Run(browserCtx,
		chromedp.SetValue(`#autoFrame`, "Seventh"),
		chromedp.WaitReady(`#importAllPrints`, chromedp.ByID),
	); err != nil {
		return err
	}

	// Further actions: check_import_all_prints and load_card
	w.logger.Info("Checking 'Import All Prints' checkbox and loading card...")
	if err := w.checkImportAllPrints(browserCtx); err != nil {
		return err
	}
	w.logger.Info("Loading card...")
	if err := w.loadCard(cardData, browserCtx); err != nil {
		return err
	}

	return nil
}

func (w *worker) checkImportAllPrints(browserCtx context.Context) error {
	var checked bool
	// Check if checkbox is checked
	err := chromedp.Run(browserCtx,
		chromedp.EvaluateAsDevTools(`document.querySelector('#importAllPrints')?.checked`, &checked),
	)
	if err != nil {
		return err
	}
	if !checked {
		w.logger.Info("Checkbox 'Import All Prints' is not checked, clicking it.")
		// Click parent element of checkbox and wait until checkbox is visible again
		err = chromedp.Run(browserCtx,
			chromedp.EvaluateAsDevTools(`document.querySelector('#importAllPrints').parentElement.click()`, nil),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *worker) loadCard(cardData common.CardInfo, browserCtx context.Context) error {
	// Before entering name: remove all options from dropdown
	w.logger.Info("Removing all options from #import-index before new search...")
	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(`document.querySelectorAll('#import-index option').forEach(o => o.remove())`, nil),
	); err != nil {
		w.logger.Warnf("Could not remove options in dropdown: %v", err)
		// not a fatal error, continue
	}

	// Press tab: set focus, then send tab key as raw event, then wait for dropdown to be ready
	if err := chromedp.Run(browserCtx,
		chromedp.WaitVisible(`#import-name`, chromedp.ByID),
		chromedp.WaitReady(`#import-name`, chromedp.ByID),
		chromedp.SetValue(`#import-name`, cardData.GetName(), chromedp.ByID),
		chromedp.Focus(`#import-name`, chromedp.ByID),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.SendKeys(`#import-name`, "\t", chromedp.ByID).Do(ctx)
		}),
		// Wait until at least one option in dropdown is loaded
		chromedp.Poll(`document.querySelectorAll('#import-index option').length > 1`, nil, chromedp.WithPollingInterval(100*time.Millisecond)),
		chromedp.WaitReady(`#import-index`, chromedp.ByID),
	); err != nil {
		return err
	}

	// Build card version string
	cardVersion := cardData.GetFullName()

	// Query all options in dropdown
	var optionTexts []string
	var optionValues []string
	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(`Array.from(document.querySelectorAll('#import-index option')).map(o => o.textContent.trim())`, &optionTexts),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('#import-index option')).map(o => o.value)`, &optionValues),
	); err != nil {
		return err
	}

	// Find matching option (also check for exact match after trim)
	var valueToSelect string
	for i, text := range optionTexts {
		if text == cardVersion {
			valueToSelect = optionValues[i]
			break
		}
	}
	if valueToSelect == "" {
		// Try whitespace-insensitive search
		for i, text := range optionTexts {
			if strings.TrimSpace(text) == strings.TrimSpace(cardVersion) {
				valueToSelect = optionValues[i]
				break
			}
		}
	}

	if valueToSelect != "" {
		// Select option and wait for dropdown to be ready again
		if err := chromedp.Run(browserCtx,
			chromedp.SetAttributeValue(fmt.Sprintf(`#import-index option[value="%s"]`, valueToSelect), "selected", "true"),
			chromedp.SetValue(`#import-index`, valueToSelect),
			chromedp.WaitReady(`#import-index`, chromedp.ByID),
		); err != nil {
			return err
		}
	} else {
		w.logger.Warnf("Warning: No matching card version found: %s", cardVersion)
	}

	return nil
}
