package cardconjurer

import (
	"cardconjurer-automation/pkg/common"
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

func (w *worker) saveCard(card common.CardInfo, browserCtx context.Context) error {
	log.Println("Speichere Karte...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("konnte Home-Verzeichnis nicht bestimmen: %v", err)
	}

	// Erwarteter Dateiname (immer mit einfachem Apostroph)
	filename := fmt.Sprintf("%s.png", card.GetName())
	downloadPath := path.Join(homeDir, "Downloads", filename)
	altFilename := strings.ReplaceAll(filename, "'", "’")
	altDownloadPath := path.Join(homeDir, "Downloads", altFilename)
	targetPath := path.Join(w.config.OutputCardsFolder, fmt.Sprintf("%s_%s.png", w.config.ProjectName, card.GetSanitizedName()))

	// Vor dem Download: Lösche ggf. existierende Datei im Download-Ordner (beide Varianten)
	if _, err := os.Stat(downloadPath); err == nil {
		log.Printf("Lösche existierende Datei im Download-Ordner: %s", downloadPath)
		if err := os.Remove(downloadPath); err != nil {
			return fmt.Errorf("Fehler beim Löschen der alten Download-Datei: %v", err)
		}
	}
	if altDownloadPath != downloadPath {
		if _, err := os.Stat(altDownloadPath); err == nil {
			log.Printf("Lösche existierende Datei im Download-Ordner: %s", altDownloadPath)
			if err := os.Remove(altDownloadPath); err != nil {
				return fmt.Errorf("Fehler beim Löschen der alten alternativen Download-Datei: %v", err)
			}
		}
	}

	// Klick auf Download-Button
	if err := chromedp.Run(browserCtx,
		chromedp.Click(`h3.download[onclick*="downloadCard"]`),
	); err != nil {
		return err
	}

	// Warte auf die Datei im Download-Ordner (beide Varianten prüfen)
	timeout := time.After(20 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var foundPath string
	for foundPath == "" {
		select {
		case <-timeout:
			return fmt.Errorf("timeout beim Warten auf Download: %s oder %s", downloadPath, altDownloadPath)
		case <-ticker.C:
			if _, err := os.Stat(downloadPath); err == nil {
				foundPath = downloadPath
				log.Printf("Karte gespeichert: %s", downloadPath)
			}
			if altDownloadPath != downloadPath {
				if _, err := os.Stat(altDownloadPath); err == nil {
					foundPath = altDownloadPath
					log.Printf("Karte gespeichert (typografisches Apostroph): %s", altDownloadPath)
				}
			}
		}
	}

	if foundPath == "" {
		return fmt.Errorf("Konnte keine heruntergeladene Datei finden: %s oder %s", downloadPath, altDownloadPath)
	}

	// Datei ins Zielverzeichnis verschieben (immer mit einfachem Apostroph im Zielnamen)
	log.Printf("Verschiebe Datei nach: %s", targetPath)
	err = os.Rename(foundPath, targetPath)
	if err != nil {
		// Fallback: Kopieren und Löschen, falls Rename fehlschlägt (z.B. über Dateisystemgrenzen)
		input, errOpen := os.Open(foundPath)
		if errOpen != nil {
			return fmt.Errorf("Fehler beim Öffnen der Quelldatei: %v", errOpen)
		}
		defer input.Close()

		output, errCreate := os.Create(targetPath)
		if errCreate != nil {
			return fmt.Errorf("Fehler beim Erstellen der Zieldatei: %v", errCreate)
		}
		defer output.Close()

		if _, errCopy := io.Copy(output, input); errCopy != nil {
			return fmt.Errorf("Fehler beim Kopieren der Datei: %v", errCopy)
		}
		input.Close()
		output.Close()
		if errRemove := os.Remove(foundPath); errRemove != nil {
			return fmt.Errorf("Fehler beim Entfernen der Quelldatei: %v", errRemove)
		}
	}
	log.Printf("Datei erfolgreich verschoben: %s", targetPath)
	return nil
}
