package cardconjurer

import (
	"cardconjurer-automation/pkg/common"
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"os"
	"time"
)

func (w *worker) addMargin(browserCtx context.Context) error {
	// Click on the frame tab and wait for the dropdown to be visible
	log.Println("Starting margin import")
	if err := chromedp.Run(browserCtx,
		chromedp.Click(`h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="frame"]`),
		chromedp.WaitVisible(`#selectFrameGroup`, chromedp.ByID),
	); err != nil {
		return err
	}

	// Select "Margin" in the dropdown and wait for the button to be ready
	log.Println("Selecting 'Margin' in frame dropdown")
	if err := chromedp.Run(browserCtx,
		chromedp.SetValue(`#selectFrameGroup`, "Margin"),
		chromedp.WaitReady(`#addToFull`, chromedp.ByID),
	); err != nil {
		return err
	}

	// Wait for the desired image element to load
	log.Println("Waiting for margin image element...")
	if err := chromedp.Run(browserCtx,
		chromedp.WaitReady(`img[src="/img/frames/margins/blackBorderExtensionThumb.png"]`),
	); err != nil {
		return err
	}

	log.Println("Clicking 'addToFull' button...")
	if err := chromedp.Run(browserCtx,
		chromedp.Click(`#addToFull`),
	); err != nil {
		return err
	}

	// After clicking 'addToFull': wait for the canvas element (#previewCanvas) to change.
	// Since the canvas cannot be directly compared, you can observe e.g. the size, an attribute or a hash of the image content.
	// Here: Read a DataURL snapshot before the click and wait until it changes.

	log.Println("Waiting for canvas to update after 'addToFull'...")
	var oldDataURL string
	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(`document.getElementById('previewCanvas')?.toDataURL()`, &oldDataURL),
	); err != nil {
		log.Printf("Could not read canvas DataURL: %v", err)
		// not a fatal error, continue
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
			log.Printf("Timeout or error while waiting for canvas update: %v", err)
		} else {
			log.Printf("Canvas updated.")
		}
	} else {
		log.Println("No canvas DataURL found, cannot wait for update.")
	}

	log.Println("Margin import finished")
	return nil
}

func (w *worker) replaceArtwork(card common.CardInfo, browserCtx context.Context) error {
	if w.config.InputArtworkFolder == "" {
		//return nil
	}

	// Click on the artwork tab and wait for the file input to be visible
	log.Println("Starting artwork import")
	inputSelector := `input[type="file"][accept*=".png"][data-dropfunction="uploadArt"]`
	err := w.openTab(
		browserCtx,
		"art",
		inputSelector,
	)
	if err != nil {
		return err
	}

	// Check if a matching PNG file exists in the artwork folder
	filename := fmt.Sprintf("%s.png", card.GetName())
	filepath := fmt.Sprintf("%s/%s", w.config.InputArtworkFolder, filename)
	if _, err := os.Stat(filepath); err != nil {
		if os.IsNotExist(err) {
			log.Printf("Artwork file not found: %s", filepath)
			return nil
		}

		return err
	}

	log.Printf("Artwork file found: %s", filepath)
	// Set the file path as value for the file input
	if err := chromedp.Run(browserCtx,
		chromedp.SetUploadFiles(inputSelector, []string{filepath}),
	); err != nil {
		log.Printf("Error setting artwork file: %v", err)
		return err
	}
	log.Printf("Artwork file %s set successfully.", filepath)

	return nil
}

func (w *worker) removeSetSymbol(browserCtx context.Context) error {
	//buttonSelector := `button.input.margin-bottom[onclick*="removeSetSymbol()"]`
	buttonSelector := `#creator-menu-setSymbol > div:nth-child(3) > button`

	err := w.openTab(browserCtx, "setSymbol", buttonSelector)
	if err != nil {
		return err
	}

	// Click the button to remove the set symbol
	if err := chromedp.Run(browserCtx,
		chromedp.Click(buttonSelector),
		chromedp.Sleep(250*time.Millisecond),
	); err != nil {
		return err
	}

	return nil
}
