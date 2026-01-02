package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"

	"github.com/scipunch/myfeed/fetcher/types"
)

const (
	maxPhotoSize = 500 * 1024 * 1024 // 500MB max file size
)

// downloadPhoto downloads a photo from Telegram and returns a MediaAttachment
// Photos are saved with a filename based on the message GUID to ensure uniqueness
func downloadPhoto(ctx context.Context, client *telegram.Client, photo *tg.Photo, messageGUID string, tmpDir string) (types.MediaAttachment, error) {
	var attachment types.MediaAttachment
	attachment.Type = "photo"

	// Find the largest photo size
	var largestSize *tg.PhotoSize
	var maxPixels int

	for _, sizeClass := range photo.Sizes {
		switch size := sizeClass.(type) {
		case *tg.PhotoSize:
			pixels := size.W * size.H
			if pixels > maxPixels {
				maxPixels = pixels
				largestSize = size
			}
		case *tg.PhotoSizeProgressive:
			// Progressive sizes - use the dimensions from the photo size
			pixels := size.W * size.H
			if pixels > maxPixels {
				maxPixels = pixels
				// Convert to PhotoSize for downloading
				largestSize = &tg.PhotoSize{
					Type: size.Type,
					W:    size.W,
					H:    size.H,
				}
			}
		}
	}

	if largestSize == nil {
		return attachment, fmt.Errorf("no suitable photo size found")
	}

	// Check file size
	// Note: Telegram doesn't always provide size info for photos, so we'll download and check
	attachment.Width = largestSize.W
	attachment.Height = largestSize.H

	// Create unique filename based on message GUID and photo ID
	filename := fmt.Sprintf("photo_%s_%d.jpg", messageGUID, photo.ID)
	localPath := filepath.Join(tmpDir, filename)

	// Ensure temp directory exists
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return attachment, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create file for download
	file, err := os.Create(localPath)
	if err != nil {
		return attachment, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Create downloader
	d := downloader.NewDownloader()

	// Create input location for the photo
	location := &tg.InputPhotoFileLocation{
		ID:            photo.ID,
		AccessHash:    photo.AccessHash,
		FileReference: photo.FileReference,
		ThumbSize:     largestSize.Type,
	}

	// Download the photo
	_, err = d.Download(client.API(), location).Stream(ctx, file)
	if err != nil {
		os.Remove(localPath) // Clean up on error
		return attachment, fmt.Errorf("failed to download photo: %w", err)
	}

	// Check file size after download
	fileInfo, err := file.Stat()
	if err != nil {
		os.Remove(localPath)
		return attachment, fmt.Errorf("failed to stat downloaded file: %w", err)
	}

	if fileInfo.Size() > maxPhotoSize {
		os.Remove(localPath)
		return attachment, fmt.Errorf("photo size (%d bytes) exceeds maximum allowed size (%d bytes)", fileInfo.Size(), maxPhotoSize)
	}

	slog.Debug("photo downloaded",
		"filename", filename,
		"size", fileInfo.Size(),
		"dimensions", fmt.Sprintf("%dx%d", attachment.Width, attachment.Height))

	attachment.LocalPath = localPath
	return attachment, nil
}

// extractMediaFromMessage extracts media attachments from a Telegram message
func extractMediaFromMessage(ctx context.Context, client *telegram.Client, msg *tg.Message, messageGUID string, tmpDir string) ([]types.MediaAttachment, error) {
	var attachments []types.MediaAttachment

	if msg.Media == nil {
		return attachments, nil
	}

	switch media := msg.Media.(type) {
	case *tg.MessageMediaPhoto:
		// Single photo
		photo, ok := media.GetPhoto()
		if !ok {
			return attachments, nil
		}

		photoObj, ok := photo.(*tg.Photo)
		if !ok {
			return attachments, nil
		}

		attachment, err := downloadPhoto(ctx, client, photoObj, messageGUID, tmpDir)
		if err != nil {
			slog.Warn("failed to download photo", "error", err, "message_id", msg.ID)
			// Return attachment with error in caption to display as alt text
			attachment.Type = "photo"
			attachment.Caption = fmt.Sprintf("Error downloading photo: %s", err.Error())
			attachments = append(attachments, attachment)
			return attachments, nil
		}

		attachments = append(attachments, attachment)

	case *tg.MessageMediaDocument:
		// Could be photo, video, or other document
		// For now, we only handle photos (ignore videos per requirements)
		doc, ok := media.GetDocument()
		if !ok {
			return attachments, nil
		}

		document, ok := doc.(*tg.Document)
		if !ok {
			return attachments, nil
		}

		// Check if it's an image by MIME type
		isImage := false
		for _, attr := range document.Attributes {
			if imgAttr, ok := attr.(*tg.DocumentAttributeImageSize); ok {
				// This is an image
				isImage = true
				_ = imgAttr // Will use dimensions later
				break
			}
		}

		if !isImage {
			// Not an image, skip (could be video, document, etc.)
			return attachments, nil
		}

		// Skip for now - MessageMediaDocument requires different handling
		// We'll focus on MessageMediaPhoto which covers most cases
		slog.Debug("skipping document media (not implemented yet)", "message_id", msg.ID)
	}

	return attachments, nil
}
