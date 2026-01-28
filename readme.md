# URL Shortener

A simple URL shortening service that converts long URLs into short, shareable links.

## Features

- Convert long URLs to short codes
- Redirect short links to original URLs
- Easy-to-use API
- Fast and reliable

## Installation

```bash
git clone <repository-url>
cd url-shortener
go run main.go
```

### API Endpoints

- `POST /shorten` - Create a short URL
- `GET /:code` - Redirect to original URL

## Example

```bash
curl -X POST http://localhost:3000/shorten \
    -H "Content-Type: application/json" \
    -d '{"url":"https://example.com/very/long/url"}'
```

## Technologies

- Golang
- PostgreSQL

## License

MIT
