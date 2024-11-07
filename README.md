# Drive Downloader

## Overview

**Drive Downloader** is a Go-based tool that allows you to interact with Google Drive via the Google Drive API. This project enables users (or service accounts) to download files and folders from Google Drive. It provides a simple command-line interface (CLI) for authentication, file listing, and file downloading from a user's Google Drive or a Google Drive shared with a service account.

This tool supports **OAuth2** authentication for user accounts and **service account** authentication for automated access. It is ideal for automating downloads or managing files in Google Drive.

## Features

- **Download Files/Folders** from Google Drive.
- **Service Account Authentication** for automated scripts and background processes.
- **OAuth2 Authentication** for user-based access to private folders/files.
- **File and Folder Listing** with the ability to filter by file type, name, and other metadata.
- **Error Handling** for permission and access issues.

## Prerequisites

Before using this tool, you need the following:

- **Go 1.18+** installed on your system. You can check the version by running `go version`.
- A **Google Cloud Project** with **Google Drive API** enabled.
- **Service Account** credentials or **OAuth2** client credentials.
- Your **Google Drive folder** or **files** must be shared with the service account or your Google account for access.

## Setting Up the Project

### 1. Clone the Repository

Clone this repository to your local machine:

```bash
git clone https://github.com/your-username/drive-downloader.git
cd drive-downloader
```

### 2. Install Dependencies

Install the required dependencies for the Google Drive API:

```bash
go mod tidy
```

### 3. Service Account Configuration

For Service Account Authentication, follow these steps:

1. Go to the Google Cloud Console.
2. Create a new project (or use an existing one).
3. Enable the Google Drive API for your project.
4. Create a Service Account with Viewer or Editor access to the desired files/folders.
5. Download the Service Account JSON credentials file and place it in your project directory.
6. Ensure that the folder you wish to access is shared with the Service Account email (found in the JSON credentials file under "client_email").

**Example Service Account JSON:**

```mdx
<pre>{`
{
  "type": "service_account",
  "project_id": "drive-downloader-441020",
  "private_key_id": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "private_key": "-----BEGIN PRIVATE KEY-----\nxxxxxxxxxxxx\n-----END PRIVATE KEY-----\n",
  "client_email": "your-service-account@drive-downloader-441020.iam.gserviceaccount.com",
  "client_id": "xxxxxxxxxxxxxxxxxxxxx",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_secret": "xxxxxxxxxxxx"
}`}</pre>
```
## 4. OAuth2 Authentication (Optional)

If you wish to authenticate as a user (for personal Google Drive access), you will need to set up OAuth2 credentials:

1. In the Google Cloud Console, go to APIs & Services > Credentials.
2. Create OAuth 2.0 credentials.
3. Download the OAuth2 JSON client secrets file.
4. Place the file in your project directory and rename it to `client_secret.json`.

## 5. Environment Variables

If you're using Service Account Authentication, set the path to your service account JSON file via the environment variable `GOOGLE_APPLICATION_CREDENTIALS`:

```bash
export GOOGLE_APPLICATION_CREDENTIALS="path/to/your/service-account.json"
```

If you're using OAuth2, ensure that the `client_secret.json` is present in the working directory.
```

## Usage

1. **Download a File**  
To download a file, use the following command:

```bash
go run drive_download_poc.go --file-id=YOUR_FILE_ID --destination=PATH_TO_SAVE
```

Where:  
- `YOUR_FILE_ID` is the Google Drive file ID (found in the URL of the file on Google Drive).  
- `PATH_TO_SAVE` is the local path where you want to save the file.

2. **Download a Folder**  
To download a folder and its contents, use:

```bash
go run drive_download_poc.go --folder-id=YOUR_FOLDER_ID --destination=PATH_TO_SAVE
```

Where:  
- `YOUR_FOLDER_ID` is the Google Drive folder ID (also found in the URL).  
- `PATH_TO_SAVE` is the local directory where you want the folder contents saved.

3. **Share a Folder with a Service Account**  
You can also share a folder with a service account programmatically. Here’s an example:

```go
func shareFolderWithServiceAccount(serviceAccountEmail string, folderID string) {
    // Use the Drive API to share the folder
    permission := &drive.Permission{
        Type:         "user",
        Role:         "reader", // Or "writer"
        EmailAddress: serviceAccountEmail,
    }

    _, err := srv.Permissions.Create(folderID, permission).Do()
    if err != nil {
        log.Fatalf("Error sharing folder: %v", err)
    }
    fmt.Println("Folder shared successfully!")
}
```

4. **Service Account Email**  
The service account email can be found in the `client_email` field of the service account JSON. This email must be granted access to the folder you want to download.

### Example Output  
When the program runs successfully, you should see output like:

```bash
Successfully downloaded file: example.txt
```

If there’s an error (e.g., access denied), you will receive an error message like:

```bash
Failed to download file: permission denied
```

### Error Handling  
The tool includes error handling for common issues, such as:
- **Permission denied:** If the service account or user doesn't have access to the requested file/folder.
- **File not found:** If the file or folder ID is incorrect or doesn't exist.
- **Invalid credentials:** If the credentials file is missing or incorrectly configured.

## Contributing  
If you would like to contribute to this project:
- Fork the repository.
- Create a new branch (`git checkout -b feature-branch`).
- Make your changes.
- Commit your changes (`git commit -am 'Add new feature'`).
- Push to the branch (`git push origin feature-branch`).
- Create a new Pull Request.

## License  
This project is licensed under the MIT License - see the LICENSE file for details.

## Contact  
For any issues or inquiries, please open an issue on GitHub or contact [your-email@example.com].

> **Note:** The service account email `your-service-account@drive-downloader-441020.iam.gserviceaccount.com` and the credentials used here are examples. You need to replace them with the credentials generated for your project.
### Explanation of Sections:
1. **Overview**: Describes the purpose of the tool and its functionality.
2. **Prerequisites**: Lists the tools and setup required to run the project.
3. **Setting Up the Project**: Guides users through the process of setting up the project, including cloning, installing dependencies, and configuring the service account or OAuth2 credentials.
4. **Usage**: Provides instructions on how to use the tool for downloading files and folders.
5. **Service Account Configuration**: Explains how to configure a service account and authenticate using the credentials JSON.
6. **Error Handling**: Describes possible errors users might encounter and how the tool handles them.
7. **Contributing**: Provides a simple guide on how others can contribute to the project.

This README should provide a comprehensive guide for using your `drive-downloader` tool, making it easy for others to set up and use in their own projects.
