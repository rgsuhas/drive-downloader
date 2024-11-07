package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// ExtractFolderID extracts the Google Drive folder ID from a folder link.
func ExtractFolderID(link string) (string, error) {
	re := regexp.MustCompile(`folders/([a-zA-Z0-9-_]+)`)
	match := re.FindStringSubmatch(link)
	if len(match) < 2 {
		return "", fmt.Errorf("invalid Google Drive folder link")
	}
	return match[1], nil
}

// GoogleDriveClient holds the Google Drive service and related configurations.
type GoogleDriveClient struct {
	Service *drive.Service
}

// NewGoogleDriveClient initializes a Google Drive client using service account credentials.
func NewGoogleDriveClient(credentialsFilePath string) (*GoogleDriveClient, error) {
	ctx := context.Background()
	creds, err := os.ReadFile(credentialsFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	config, err := google.CredentialsFromJSON(ctx, creds, drive.DriveReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials from JSON: %w", err)
	}

	svc, err := drive.NewService(ctx, option.WithCredentials(config))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %w", err)
	}

	return &GoogleDriveClient{Service: svc}, nil
}

// ListFiles lists files within a specified Google Drive folder.
func (c *GoogleDriveClient) ListFiles(folderID string) ([]*drive.File, error) {
	query := fmt.Sprintf("'%s' in parents", folderID)
	fileList, err := c.Service.Files.List().Q(query).Fields("files(id, name)").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve files: %w", err)
	}
	return fileList.Files, nil
}

// DownloadFolder downloads files from a Google Drive folder to the specified path.
func (c *GoogleDriveClient) DownloadFolder(folderID, downloadPath string) error {
	files, err := c.ListFiles(folderID)
	if err != nil {
		return err
	}

	for _, file := range files {
		fmt.Printf("Downloading file: %s\n", file.Name)
		if err := c.downloadFile(file.Id, filepath.Join(downloadPath, file.Name)); err != nil {
			return err
		}
	}
	return nil
}

// downloadFile downloads a file by its ID and saves it to the specified path.
func (c *GoogleDriveClient) downloadFile(fileID, filePath string) error {
	resp, err := c.Service.Files.Get(fileID).Download()
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}
	return nil
}

func main() {
	// Specify the Google Drive folder link, credentials file, and download path.
	driveFolderLink := "https://drive.google.com/drive/folders/1z_wZgiQXlATYiRyv0BQo28eNBQrAuOnq?usp=sharing"
	credentialsFilePath := "C://Users/suhas/Downloads/drive-downloader-441020-55befecc38fc.json"
	downloadPath := "C://Users/suhas/Downloads/drive-downloader"

	// Extract folder ID from the link.
	folderID, err := ExtractFolderID(driveFolderLink)
	if err != nil {
		log.Fatalf("Failed to extract folder ID: %v", err)
	}

	// Initialize Google Drive client.
	driveClient, err := NewGoogleDriveClient(credentialsFilePath)
	if err != nil {
		log.Fatalf("Failed to initialize Google Drive client: %v", err)
	}

	// Ensure the download path exists.
	if err := os.MkdirAll(downloadPath, os.ModePerm); err != nil {
		log.Fatalf("Failed to create download directory: %v", err)
	}

	// List and download files from the specified folder.
	files, err := driveClient.ListFiles(folderID)
	if err != nil {
		log.Fatalf("Failed to list files: %v", err)
	}
	fmt.Println("Files in Google Drive folder:")
	for _, file := range files {
		fmt.Println(file.Name)
	}

	// Download files to the specified directory.
	if err := driveClient.DownloadFolder(folderID, downloadPath); err != nil {
		log.Fatalf("Failed to download folder: %v", err)
	}

	fmt.Println("Download completed successfully.")
}
