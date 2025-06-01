# Scrapper Backend

This is the backend service for the Scrapper application, written in Go. It handles RSS feed scraping, storage in MongoDB, and provides a RESTful API for the frontend.

## Features
- Fetches and parses RSS feeds from specified URLs.
- Stores feed items in MongoDB with deduplication based on links.
- Provides endpoints for creating, reading, updating, and deleting feeds and their items.
- Supports user authentication via Firebase.

## Prerequisites
- Go (1.18 or later)
- MongoDB (local or remote instance)
- Firebase project with Authentication enabled

## Setup
1. Clone the repository:
   ```
   git clone https://github.com/kwabena369/scrapper.git
   cd scrapper
   ```
2. Install dependencies:
   ```
   go mod tidy
   ```
3. Set up environment variables:
   - Create a `.env` file with:
     ```
   PORT=8080
   MONGO_URI=
   FIREBASE_CRED_PATH=
<!-- gmail credential for sending email notifications -->
   EMAIL_PASS=
   EMAIL_USER=
     ```
   - Obtain Firebase credentials from your Firebase Console (Service Account).
4. Run the application:
   ```
   go run cmd/api/main.go
   ```
   The server will start on `http://localhost:8080`.

## API Endpoints
- `GET /v1/feeds`: List all feeds.
- `GET /v1/feeds/:id`: Get a specific feed.
- `POST /v1/feeds`: Create a new feed.
- `PUT /v1/feeds/:id`: Update a feed.
- `DELETE /v1/feeds/:id`: Delete a feed.
- `GET /v1/feeds/:id/items`: Get items for a feed.
- `POST /v1/feeds/:id/scrape`: Trigger scraping for a feed.

## Development
- Use `go fmt` and `go vet` to maintain code quality.
- Add tests in the `internal` package using Go’s testing framework.

## Contributing
Feel free to submit issues or pull requests. Ensure code follows Go conventions.

## License
MIT License (specify if different).
`