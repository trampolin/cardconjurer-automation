package cardconjurer

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"os"
	"time"
)

func (w *worker) openBrowser(parentCtx context.Context) (context.Context, error) {
	log.Println("Starting new browser...")
	log.Printf("Creating temp directory for worker %d: %s", w.workerID, w.tempDirName)
	dir, err := os.MkdirTemp("", w.tempDirName)
	if err != nil {
		log.Printf("Error creating temp directory: %v", err)
		return nil, err
	}
	// os.RemoveAll(dir) should be handled by the caller

	// Explicitly set the download folder to the desired OutputCardsFolder
	downloadDir := w.config.OutputCardsFolder
	if downloadDir == "" {
		downloadDir = dir // fallback
	}

	log.Println("Download directory:", downloadDir)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.UserDataDir(dir),
		chromedp.Flag("headless", false),
		chromedp.Flag("download.default_directory", downloadDir),
		chromedp.Flag("download.prompt_for_download", false),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(parentCtx, opts...)
	// cancel should be handled by the caller

	taskCtx, cancel2 := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	// cancel2 should be handled by the caller

	log.Printf("Opening browser at %s and sleeping for two seconds", w.config.BaseUrl)
	if err := chromedp.Run(taskCtx,
		chromedp.Navigate(w.config.BaseUrl),
		// Wait until the document is fully loaded
		chromedp.WaitReady("body"),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		cancel2()
		cancel()
		return nil, err
	}

	return taskCtx, nil
}

func (w *worker) closeBrowser(browserCtx context.Context) {
	log.Println("Closing browser...")
	if err := chromedp.Cancel(browserCtx); err != nil {
		log.Printf("Error closing browser: %v", err)
	}
	log.Println("Browser closed")
	// Delete temp directory
	if w.tempDirName != "" {
		if err := os.RemoveAll(w.tempDirName); err != nil {
			log.Printf("Error deleting temp directory %s: %v", w.tempDirName, err)
		} else {
			log.Printf("Temp directory deleted: %s", w.tempDirName)
		}
	}
}

// openTab opens a tab by its name (e.g. "import", "frame").
// It can wait for any number of selectors after the click.
func (w *worker) openTab(ctx context.Context, tabName string, waitForSelectors ...string) error {
	selector := fmt.Sprintf(`h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="%s"]`, tabName)
	log.Printf("Opening tab: %s", tabName)
	actions := []chromedp.Action{
		chromedp.Click(selector),
	}
	for _, sel := range waitForSelectors {
		if sel != "" {
			log.Printf("Waiting for element after tab switch: %s", sel)
			actions = append(actions, chromedp.WaitVisible(sel))
		}
	}
	return chromedp.Run(ctx, actions...)
}
