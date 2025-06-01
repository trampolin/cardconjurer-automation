package cardconjurer

import (
	"cardconjurer-automation/pkg/common"
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

func (w *worker) saveCard(card common.CardInfo, browserCtx context.Context) error {
	w.logger.Info("Saving card")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %v", err)
	}

	// Expected filename (always with straight apostrophe)
	filename := fmt.Sprintf("%s.png", card.GetName())
	downloadPath := path.Join(homeDir, "Downloads", filename)
	altFilename := strings.ReplaceAll(filename, "'", "’")
	altDownloadPath := path.Join(homeDir, "Downloads", altFilename)
	targetPath := path.Join(w.config.OutputCardsFolder, fmt.Sprintf("%s_%s.png", w.config.ProjectName, card.GetSanitizedName()))

	// Before download: Delete existing file in download folder if present (both variants)
	if _, err := os.Stat(downloadPath); err == nil {
		w.logger.Infof("Deleting existing file in download folder: %s", downloadPath)
		if err := os.Remove(downloadPath); err != nil {
			return fmt.Errorf("error deleting old download file: %v", err)
		}
	}
	if altDownloadPath != downloadPath {
		if _, err := os.Stat(altDownloadPath); err == nil {
			w.logger.Infof("Deleting existing file in download folder: %s", altDownloadPath)
			if err := os.Remove(altDownloadPath); err != nil {
				return fmt.Errorf("error deleting old alternative download file: %v", err)
			}
		}
	}

	// Click download button
	if err := chromedp.Run(browserCtx,
		chromedp.Click(`h3.download[onclick*="downloadCard"]`),
	); err != nil {
		return err
	}

	// Wait for file in download folder (check both variants)
	timeout := time.After(20 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var foundPath string
	for foundPath == "" {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for download: %s or %s", downloadPath, altDownloadPath)
		case <-ticker.C:
			if _, err := os.Stat(downloadPath); err == nil {
				foundPath = downloadPath
				w.logger.Infof("Card saved: %s", downloadPath)
			}
			if altDownloadPath != downloadPath {
				if _, err := os.Stat(altDownloadPath); err == nil {
					foundPath = altDownloadPath
					w.logger.Infof("Card saved (typographic apostrophe): %s", altDownloadPath)
				}
			}
		}
	}

	if foundPath == "" {
		return fmt.Errorf("could not find downloaded file: %s or %s", downloadPath, altDownloadPath)
	}

	// Move file to target directory (always with straight apostrophe in target name)
	w.logger.Infof("Moving file to: %s", targetPath)
	err = os.Rename(foundPath, targetPath)
	if err != nil {
		// Fallback: Copy and delete if Rename fails (e.g. across filesystems)
		input, errOpen := os.Open(foundPath)
		if errOpen != nil {
			return fmt.Errorf("error opening source file: %v", errOpen)
		}
		defer input.Close()

		output, errCreate := os.Create(targetPath)
		if errCreate != nil {
			return fmt.Errorf("error creating target file: %v", errCreate)
		}
		defer output.Close()

		if _, errCopy := io.Copy(output, input); errCopy != nil {
			return fmt.Errorf("error copying file: %v", errCopy)
		}
		input.Close()
		output.Close()
		if errRemove := os.Remove(foundPath); errRemove != nil {
			return fmt.Errorf("error removing source file: %v", errRemove)
		}
	}
	w.logger.Infof("File successfully moved: %s", targetPath)
	return nil
}
