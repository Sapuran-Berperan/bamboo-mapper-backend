package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GDriveService handles file uploads to Google Drive
type GDriveService struct {
	service  *drive.Service
	folderID string
}

// NewGDriveService creates a new Google Drive service using OAuth2 credentials
// credentialsPath: path to OAuth2 client credentials JSON (from Google Cloud Console)
// tokenPath: path to the saved OAuth2 token JSON (created by running scripts/gdrive_auth.go)
// folderID: the Google Drive folder ID to upload files to
func NewGDriveService(credentialsPath, tokenPath, folderID string) (*GDriveService, error) {
	ctx := context.Background()

	// Read OAuth2 client credentials
	credBytes, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Parse credentials and create OAuth2 config
	config, err := google.ConfigFromJSON(credBytes, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Read saved token
	token, err := tokenFromFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token (run 'go run scripts/gdrive_auth.go' first): %w", err)
	}

	// Create token source that auto-refreshes and saves the new token
	tokenSource := config.TokenSource(ctx, token)

	// Get a fresh token (this will refresh if expired)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Save the refreshed token if it changed
	if newToken.AccessToken != token.AccessToken {
		if err := saveToken(tokenPath, newToken); err != nil {
			// Log but don't fail - the service can still work
			fmt.Printf("Warning: failed to save refreshed token: %v\n", err)
		}
	}

	// Create Drive service with the token source
	service, err := drive.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	return &GDriveService{
		service:  service,
		folderID: folderID,
	}, nil
}

// tokenFromFile reads a token from a file
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves a token to a file
func saveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
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
