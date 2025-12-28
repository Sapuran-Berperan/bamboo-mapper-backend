package storage

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GDriveService handles file uploads to Google Drive
type GDriveService struct {
	service  *drive.Service
	folderID string
}

// NewGDriveService creates a new Google Drive service using service account credentials
func NewGDriveService(credentialsPath, folderID string) (*GDriveService, error) {
	ctx := context.Background()

	// Use WithAuthCredentialsFile for better security validation
	service, err := drive.NewService(ctx, option.WithAuthCredentialsFile(option.ServiceAccount, credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	return &GDriveService{
		service:  service,
		folderID: folderID,
	}, nil
}

// UploadFile uploads a file to Google Drive and returns the shareable URL
func (g *GDriveService) UploadFile(file io.Reader, filename, mimeType string) (string, error) {
	// Create file metadata
	driveFile := &drive.File{
		Name:    filename,
		Parents: []string{g.folderID},
	}

	// Upload file
	createdFile, err := g.service.Files.Create(driveFile).
		Media(file).
		Fields("id, webViewLink, webContentLink").
		Do()
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Make file publicly accessible
	permission := &drive.Permission{
		Type: "anyone",
		Role: "reader",
	}
	_, err = g.service.Permissions.Create(createdFile.Id, permission).Do()
	if err != nil {
		return "", fmt.Errorf("failed to set file permissions: %w", err)
	}

	// Return direct download link
	// Format: https://drive.google.com/uc?id=FILE_ID
	return fmt.Sprintf("https://drive.google.com/uc?id=%s", createdFile.Id), nil
}

// DeleteFile deletes a file from Google Drive by its ID
func (g *GDriveService) DeleteFile(fileID string) error {
	err := g.service.Files.Delete(fileID).Do()
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}
