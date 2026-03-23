# PhotoAlbum

A single-binary photo album application designed for local network usage.

This project is implemented in Go, without front-end / back-end separation. After startup, it can be accessed directly within the LAN. Front-end assets are embedded into the executable, so deployment only requires one binary and a config file placed alongside it.

## Features

- Single binary deployment
- LAN access
- Multi-user login and isolation
- Automatic first-run initialization
- Timeline view
- Custom album management
- Click and drag-and-drop photo upload
- Automatic thumbnail generation
- Trash and restore
- Photo sharing
- Single photo download / batch ZIP download / full album ZIP download
- Anonymous shared-photo download
- Light / dark theme
- Responsive support for desktop and mobile

## Implemented Features

### Core Capabilities

- Written in Go without front-end/back-end separation
- Front-end assets embedded into the binary
- Builds on Linux, macOS, and Windows
- Auto-generates config on first launch
- Creates a default administrator during initialization

### Users and Authentication

- Multi-user support
- Users managed via config file
- `adduser` CLI command
- Passwords stored with bcrypt hash
- JWT + Cookie based session
- Login / logout

### Photo Features

- Click-to-upload
- Drag-and-drop upload
- Multi-file upload
- Upload progress display
- Retry single failed upload
- Retry all failed uploads
- EXIF timestamp extraction
- Falls back to client file modification time when EXIF is missing
- Automatic thumbnail generation
- EXIF Orientation correction
- Lightbox preview
- Previous / next navigation
- Basic metadata display

### Timeline

- Grouped by date
- Infinite scrolling
- Multi-select
- Select all by date
- Batch delete
- Batch add to album
- Batch ZIP download for selected photos

### Albums

- Create album
- Delete album
- Album list view
- Album cover thumbnails
- Album detail page
- Date-grouped layout inside album
- Infinite scrolling inside album
- Refresh restores current album detail page
- Full album ZIP download

### Trash

- Soft delete
- Preview in trash
- Restore single photo
- Batch restore
- Permanently delete single photo
- Empty entire trash

### Sharing

- Create photo share links
- Expiration time support
- Shared-state badge on photos
- View / copy / delete share links
- Anonymous share page access
- Anonymous shared-photo download

### Downloads

- Single photo download
- Batch selected photos as ZIP
- Full album as ZIP
- Anonymous shared photo download
- Duplicate file names handled inside ZIP archives

### Front-End Experience

- Light / dark theme
- Theme preference persistence
- Desktop sidebar navigation
- Mobile drawer navigation
- Long-press action menu on mobile
- Scrollable upload queue

## Tech Stack

- Go 1.26
- SQLite (`modernc.org/sqlite`)
- JWT (`github.com/golang-jwt/jwt/v5`)
- Plain HTML / CSS / JavaScript
- Embedded front-end assets via `embed`
- Repository pattern
- SQLite WAL mode

## Quick Start

### 1. Build

```bash
go build -o photoalbum .
```

### 2. First Run

```bash
./photoalbum
```

If there is no `config.json` in the executable directory, the app will enter the initialization wizard and prompt for:

- Server port
- Photo storage path
- Default username
- Default password

The config file will be created automatically after initialization.

### 3. Login

Open in your browser:

```text
http://127.0.0.1:8080
```

Then sign in with the user created during initialization.

## Commands

### Add User

```bash
./photoalbum adduser
```

Follow the prompts to create a new user, which will be saved into the config file.

## Example Configuration

```json
{
  "port": 8080,
  "storage_path": "./photos",
  "jwt_secret": "your-random-secret",
  "users": [
    {
      "username": "admin",
      "password_hash": "$2a$10$..."
    }
  ]
}
```

### Field Description

| Field | Description |
|---|---|
| `port` | HTTP server port |
| `storage_path` | Directory for photos and database |
| `jwt_secret` | JWT signing secret |
| `users` | User list |
| `users[].username` | Username |
| `users[].password_hash` | bcrypt password hash |

## Project Structure

```text
.
├── main.go
├── embed.go
├── internal/
│   ├── config/
│   ├── image/
│   ├── server/
│   ├── service/
│   └── storage/
├── web/
│   └── static/
├── README.md
└── README.en.md
```

### Directory Overview

| Directory | Responsibility |
|---|---|
| `internal/config` | Config loading, init wizard, user management |
| `internal/image` | Metadata, thumbnails, orientation handling |
| `internal/server` | HTTP routes and handlers |
| `internal/service` | Business logic layer |
| `internal/storage` | Repository abstraction and SQLite implementation |
| `web/static` | Front-end static assets |

## Core Interaction Flow

### Timeline

- Default entry view
- Photos grouped by shooting date
- Supports right-click / long-press actions
- Supports multi-select, batch delete, batch download, and batch add to albums

### Albums

- Create custom albums
- Photos can belong to multiple albums
- Album detail pages can be restored after refresh
- Album covers display thumbnails automatically

### Trash

- Deleted photos go to trash
- Supports preview, restore, batch restore, permanent delete, and empty trash
- Thumbnails are still available for trashed photos

### Sharing

- Create share links for single photos
- Supports expiration time
- Anonymous share page access
- Anonymous download for shared photos

## Testing

Run all tests:

```bash
go test ./...
```

Build verification:

```bash
go build ./...
```

## Not Implemented Yet / Future Plans

- Video support
- Search feature
- Tagging system
- Web-based user management
- More complete video thumbnails and transcoding
- Better `.gitignore`

## Notes

This project is designed for intranet usage, with current priorities focused on:

- Simple single-binary deployment
- Easy LAN access
- A complete and practical core photo-album workflow

Future extensions can continue building on the current Repository and Service layering to gradually add video, search, and more advanced management features.
