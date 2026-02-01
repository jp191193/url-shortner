# URL Shortener (Go + Rediss + PostgreSQL)

A fast, clean, and production-style **URL Shortner** built using **Go**, **Redis**, and **PostgreSQL**, following **Clean Architecture** principles.

This project demonstrates:

- High-performance Go backend.
- Redis caching (cache-aside pattern).
- PostgreSQL persistent storage.
- Modular, scalable, test-friendly architecture.
- Professional folder structure.

Perfect for learning Go, improving system design skills, or showcasing backend engineering expertise.

---

## ⭐ Features

- 🔗 Generate short URLs
- 🚀 Fast redirects using redis cache

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
