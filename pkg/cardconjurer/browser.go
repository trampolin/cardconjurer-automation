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
	log.Println("Starte neuen Browser...")
	log.Printf("Erstelle Temp-Verzeichnis für Worker %d: %s", w.workerID, w.tempDirName)
	dir, err := os.MkdirTemp("", w.tempDirName)
	if err != nil {
		log.Printf("Fehler beim Erstellen des Temp-Verzeichnisses: %v", err)
		return nil, err
	}
	// os.RemoveAll(dir) sollte vom Aufrufer übernommen werden

	// Setze den Download-Ordner explizit auf das gewünschte OutputCardsFolder
	downloadDir := w.config.OutputCardsFolder
	if downloadDir == "" {
		downloadDir = dir // Fallback
	}

	log.Println("Download Dir:", downloadDir)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.UserDataDir(dir),
		chromedp.Flag("headless", false),
		chromedp.Flag("download.default_directory", downloadDir),
		chromedp.Flag("download.prompt_for_download", false),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(parentCtx, opts...)
	// cancel sollte vom Aufrufer übernommen werden

	taskCtx, cancel2 := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	// cancel2 sollte vom Aufrufer übernommen werden

	log.Printf("Opening browser at %s and sleep for two seconds", w.config.BaseUrl)
	if err := chromedp.Run(taskCtx,
		chromedp.Navigate(w.config.BaseUrl),
		// Warte, bis das Dokument vollständig geladen ist
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
	log.Println("Schließe Browser...")
	if err := chromedp.Cancel(browserCtx); err != nil {
		log.Printf("Error closing browser: %v", err)
	}
	log.Println("Browser closed")
	// Temp-Verzeichnis löschen
	if w.tempDirName != "" {
		if err := os.RemoveAll(w.tempDirName); err != nil {
			log.Printf("Fehler beim Löschen des Temp-Verzeichnisses %s: %v", w.tempDirName, err)
		} else {
			log.Printf("Temp-Verzeichnis gelöscht: %s", w.tempDirName)
		}
	}
}

// openTab öffnet ein Tab anhand des Tab-Namens (z.B. "import", "frame").
// Es kann auf beliebig viele Selektoren nach dem Klick gewartet werden.
func (w *worker) openTab(ctx context.Context, tabName string, waitForSelectors ...string) error {
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
