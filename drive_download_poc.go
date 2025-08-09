package main

import (
	"context"
    "errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
    "strings"

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

    // Create token source from JSON and pass as client option.
    if _, err := google.CredentialsFromJSON(ctx, creds, drive.DriveReadonlyScope); err != nil {
        return nil, fmt.Errorf("failed to create credentials from JSON: %w", err)
    }

    svc, err := drive.NewService(ctx, option.WithCredentialsJSON(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %w", err)
	}

	return &GoogleDriveClient{Service: svc}, nil
}

// ListChildren lists both files and folders directly under a specified Google Drive folder.
func (c *GoogleDriveClient) ListChildren(folderID string, includeAllDrives bool) ([]*drive.File, error) {
    query := fmt.Sprintf("'%s' in parents and trashed=false", folderID)
    var results []*drive.File
    pageToken := ""
    for {
        call := c.Service.Files.List().Q(query).
            Fields("nextPageToken, files(id, name, mimeType, size, md5Checksum)").
            PageSize(1000)
        if includeAllDrives {
            call = call.SupportsAllDrives(true).IncludeItemsFromAllDrives(true)
        }
        if pageToken != "" {
            call = call.PageToken(pageToken)
        }
        fileList, err := call.Do()
        if err != nil {
            return nil, fmt.Errorf("failed to retrieve files: %w", err)
        }
        results = append(results, fileList.Files...)
        if fileList.NextPageToken == "" {
            break
        }
        pageToken = fileList.NextPageToken
    }
    return results, nil
}

// DownloadFolderRecursive downloads all files and subfolders of a Google Drive folder to the specified path.
func (c *GoogleDriveClient) DownloadFolderRecursive(folderID, downloadPath string, includeAllDrives, skipExisting bool) error {
    if err := os.MkdirAll(downloadPath, os.ModePerm); err != nil {
        return fmt.Errorf("failed to create directory %s: %w", downloadPath, err)
    }

    children, err := c.ListChildren(folderID, includeAllDrives)
    if err != nil {
        return err
    }

    for _, child := range children {
        if child.MimeType == "application/vnd.google-apps.folder" {
            subDir := filepath.Join(downloadPath, child.Name)
            fmt.Printf("Entering folder: %s\n", subDir)
            if err := c.DownloadFolderRecursive(child.Id, subDir, includeAllDrives, skipExisting); err != nil {
                return err
            }
            continue
        }

        destFilePath := filepath.Join(downloadPath, child.Name)
        if skipExisting {
            if info, err := os.Stat(destFilePath); err == nil && !info.IsDir() {
                fmt.Printf("Skipping existing file: %s\n", destFilePath)
                continue
            }
        }
        // Google-native files need exporting
        if strings.HasPrefix(child.MimeType, "application/vnd.google-apps") {
            fmt.Printf("Exporting Google document: %s (%s)\n", child.Name, child.MimeType)
            if err := c.exportGoogleFile(child, downloadPath, includeAllDrives); err != nil {
                return err
            }
            continue
        }

        fmt.Printf("Downloading file: %s\n", destFilePath)
        if err := c.downloadFile(child.Id, destFilePath, includeAllDrives); err != nil {
            return err
        }
    }
    return nil
}

// downloadFile downloads a file by its ID and saves it to the specified path.
func (c *GoogleDriveClient) downloadFile(fileID, filePath string, includeAllDrives bool) error {
    // Ensure parent directory exists in case caller didn't create it
    if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
        return fmt.Errorf("failed to create parent directory: %w", err)
    }

    getCall := c.Service.Files.Get(fileID)
    if includeAllDrives {
        getCall = getCall.SupportsAllDrives(true)
    }
    resp, err := getCall.Download()
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
    // CLI flags
    type config struct {
        credentialsPath   string
        folderArg         string // can be folder ID or a full link
        destinationPath   string
        includeAllDrives  bool
        skipExistingFiles bool
    }

    // Minimal flag parsing without external deps
    cfg := config{}
    for i := 1; i < len(os.Args); i++ {
        arg := os.Args[i]
        switch arg {
        case "-credentials", "--credentials":
            i++
            if i >= len(os.Args) {
                log.Fatal("-credentials requires a path value")
            }
            cfg.credentialsPath = os.Args[i]
        case "-folder", "--folder":
            i++
            if i >= len(os.Args) {
                log.Fatal("-folder requires an ID or link value")
            }
            cfg.folderArg = os.Args[i]
        case "-dest", "--dest":
            i++
            if i >= len(os.Args) {
                log.Fatal("-dest requires a path value")
            }
            cfg.destinationPath = os.Args[i]
        case "-all-drives", "--all-drives":
            cfg.includeAllDrives = true
        case "-skip-existing", "--skip-existing":
            cfg.skipExistingFiles = true
        case "-h", "--help":
            printUsage()
            return
        default:
            // Allow positional: folder then dest
            if cfg.folderArg == "" {
                cfg.folderArg = arg
            } else if cfg.destinationPath == "" {
                cfg.destinationPath = arg
            } else {
                log.Fatalf("unexpected argument: %s", arg)
            }
        }
    }

    if cfg.credentialsPath == "" {
        log.Fatal("missing required -credentials path to a Service Account JSON file")
    }
    if cfg.folderArg == "" {
        log.Fatal("missing required -folder (Google Drive folder ID or link)")
    }
    if cfg.destinationPath == "" {
        cwd, _ := os.Getwd()
        cfg.destinationPath = cwd
    }

    folderID := cfg.folderArg
    if strings.Contains(cfg.folderArg, "drive.google.com") || strings.HasPrefix(cfg.folderArg, "http://") || strings.HasPrefix(cfg.folderArg, "https://") {
        id, err := ExtractFolderID(cfg.folderArg)
        if err != nil {
            log.Fatalf("Failed to extract folder ID: %v", err)
        }
        folderID = id
    }

    if err := validateFolderID(folderID); err != nil {
        log.Fatalf("Invalid folder ID: %v", err)
    }

    // Initialize Google Drive client.
    driveClient, err := NewGoogleDriveClient(cfg.credentialsPath)
    if err != nil {
        log.Fatalf("Failed to initialize Google Drive client: %v", err)
    }

    fmt.Printf("Downloading Drive folder %s to %s\n", folderID, cfg.destinationPath)
    if err := driveClient.DownloadFolderRecursive(folderID, cfg.destinationPath, cfg.includeAllDrives, cfg.skipExistingFiles); err != nil {
        log.Fatalf("Failed to download folder: %v", err)
    }

    fmt.Println("Download completed successfully.")
}

func validateFolderID(id string) error {
    if id == "" {
        return errors.New("empty ID")
    }
    // Basic sanity: Drive IDs are typically URL-safe base64-ish; keep it simple here
    if strings.ContainsAny(id, "/?&") {
        return errors.New("ID contains invalid characters")
    }
    return nil
}

func printUsage() {
    fmt.Println("Usage: drive-downloader -credentials <path> -folder <id|link> [-dest <path>] [--all-drives] [--skip-existing]")
    fmt.Println()
    fmt.Println("Positional form also supported: drive-downloader -credentials <path> <id|link> [dest]")
}

// exportGoogleFile exports Google-native document formats (Docs/Sheets/Slides) to common formats.
func (c *GoogleDriveClient) exportGoogleFile(file *drive.File, destDir string, includeAllDrives bool) error {
    exportMap := map[string]struct {
        mime string
        ext  string
    }{
        // Docs → PDF
        "application/vnd.google-apps.document":      {mime: "application/pdf", ext: ".pdf"},
        // Sheets → CSV or XLSX; prefer XLSX for multi-sheet support
        "application/vnd.google-apps.spreadsheet":  {mime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ext: ".xlsx"},
        // Slides → PDF
        "application/vnd.google-apps.presentation": {mime: "application/pdf", ext: ".pdf"},
        // Drawings → PNG
        "application/vnd.google-apps.drawing":      {mime: "image/png", ext: ".png"},
    }

    rule, ok := exportMap[file.MimeType]
    if !ok {
        // For unknown google-apps types, fallback to PDF when possible
        rule = struct{ mime, ext string }{mime: "application/pdf", ext: ".pdf"}
    }

    safeName := file.Name
    // Avoid duplicate extensions
    if !strings.HasSuffix(strings.ToLower(safeName), strings.ToLower(rule.ext)) {
        safeName += rule.ext
    }
    destPath := filepath.Join(destDir, safeName)

    // Export call does not expose SupportsAllDrives; Shared Drives are handled by permission on the file ID
    resp, err := c.Service.Files.Export(file.Id, rule.mime).Download()
    if err != nil {
        return fmt.Errorf("failed to export file %s: %w", file.Name, err)
    }
    defer resp.Body.Close()

    if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }
    out, err := os.Create(destPath)
    if err != nil {
        return fmt.Errorf("failed to create file: %w", err)
    }
    defer out.Close()
    if _, err := io.Copy(out, resp.Body); err != nil {
        return fmt.Errorf("failed to write exported content: %w", err)
    }
    fmt.Printf("Exported: %s -> %s\n", file.Name, destPath)
    return nil
}
