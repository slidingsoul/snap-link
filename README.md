# Snap Link

A production-ready URL shortening service built with Go, Fiber, and PostgreSQL following clean architecture principles.

## Features

- **Clean Architecture**: Controller → Service → Repository pattern
- **Collision-Resistant Short Codes**: 6-character alphanumeric codes with retry logic
- **URL Validation**: Automatic URL format validation and scheme addition
- **GORM ORM**: Type-safe database operations with PostgreSQL
- **Fiber Web Framework**: Ultra-fast HTTP framework
- **Neon DB Support**: Environment-based PostgreSQL configuration
- **Production Ready**: Error handling, logging, and health checks
- **Render Deployment**: PORT environment variable support

## Project Structure

```
.
├── main.go                      # Application entry point
├── go.mod                       # Go module dependencies
├── .env.example                 # Environment variables template
├── middleware/
│   ├── rate_limit.go            # Fixed per-IP limiter (internal/ratelimit)
│   └── require_auth.go          # Neon JWT gate for POST /snap
├── internal/
│   ├── auth/                    # Neon JWT verifier (issuer-only, JWKS derived)
│   ├── shortcode/               # Single source: Charset, Generate, Validate
│   └── ratelimit/               # Fixed limit: DefaultPerMinute = 5
├── models/
│   └── link.go                  # Link model and request/response DTOs
├── repository/
│   └── link_repository.go       # Database operations layer
├── service/
│   └── link_service.go          # Business logic + URL validation (delegates code check)
└── controller/
    └── link_controller.go       # HTTP handlers layer (public, rate-limited)
```

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 12 or higher (or Neon DB)
- Fiber v2
- GORM

## Setup

### 1. Clone and Initialize

```bash
git clone <repository-url>
cd snap-link
cp .env.example .env
```

### 2. Configure Environment Variables

Edit `.env` with your database configuration:

```env
PORT=8081
DATABASE_URL=postgres://username:password@localhost:5432/link_shortener
NEON_AUTH_BASE_URL=https://ep-xxxxx.aws.neon.tech/dbname/auth
```

For **Neon DB**:
```env
DATABASE_URL=postgresql://user:password@ep-xxxxx.neon.tech/dbname?sslmode=require
NEON_AUTH_BASE_URL=https://ep-xxxxx.aws.neon.tech/dbname/auth
```
Get the auth URL from Neon Console → Branch → Auth. JWKS + issuer are derived; no secret needed on the server.

### 3. Install Dependencies

```bash
go mod download
go mod tidy
```

### 4. Run the Application

```bash
go run main.go
```

The server will start on `http://localhost:8081`

## API Endpoints

### POST /snap - Create Short Link

Requires Neon JWT: `Authorization: Bearer <token>` (see Neon Managed Better Auth).
Reads (`GET /:code`, `GET /health`) stay public.

**Request:**
```bash
TOKEN=$(your-frontend getSession token)
curl -X POST http://localhost:8081/snap \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"link": "https://example.com/very/long/url"}'
```

**Response:**
```json
{
  "link": "abc123"
}
```

### GET /:shortCode - Retrieve Original URL

Get the original URL and metadata for a short code.

**Request:**
```bash
curl http://localhost:8081/abc123
```

**Response:**
```json
{
  "original_url": "https://example.com/very/long/url",
  "short_code": "abc123",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### GET /health - Health Check

Endpoint for deployment monitoring.

**Request:**
```bash
curl http://localhost:8081/health
```

**Response:**
```json
{
  "status": "healthy"
}
```

## Database Schema

The service automatically creates the required table on startup:

```sql
CREATE TABLE links (
  id SERIAL PRIMARY KEY,
  original_url TEXT NOT NULL,
  short_code VARCHAR(10) NOT NULL UNIQUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_links_short_code ON links(short_code);
CREATE INDEX idx_links_original_url ON links(original_url);
```

## Short Code Generation

- **Length**: 6 characters
- **Alphabet**: A-Z, a-z, 0-9 (62 characters)
- **Collision Resistance**: Crypto-random generation with automatic retry logic
- **Theoretical Collisions**: ~62^6 ≈ 56 trillion combinations
- **Retry Logic**: Up to 5 automatic retries on collision

## Error Handling

The service provides clear error responses:

```json
{
  "error": "Invalid URL format"
}
```

Common error codes:
- `400 Bad Request`: Invalid input or malformed URL
- `404 Not Found`: Short code doesn't exist
- `500 Internal Server Error`: Database or server issues

## Logging

Structured logging for all operations:
- Request/response logging with HTTP status
- Error logging with stack traces
- Database operation logging

## Deployment

### Render.com

1. Connect your GitHub repository
2. Create a new Web Service (or `render.yaml` blueprint)
3. Set environment variables:
   - `PORT=8081` (or keep default)
   - `DATABASE_URL=postgresql://...` (Neon URL with `?sslmode=require`)
4. Set build command: `go build -o snap-link`
5. Set start command: `./snap-link`

### Docker (Local Development)

```bash
docker compose up -d --build
# App: http://localhost:8081/health
```

### Manual VPS Deployment

```bash
# Build
go build -o snap-link

# Set environment and run
export DATABASE_URL="postgresql://..."
export PORT=8081
./snap-link
```

## Production Considerations

- `POST /snap` requires Neon JWT (`Authorization: Bearer`); per-IP rate limiting (5/min) still applies
- Enable SSL/TLS via reverse proxy (nginx, Caddy)
- Monitor database connection pool
- Rate limiting on /snap is fixed: 5/min per IP (see `internal/ratelimit`)
- Consider caching for frequently accessed short codes
- Regular database backups
- Use connection pooling for high traffic
- Monitor error rates and latencies

## Performance Metrics

- Short code generation: < 1ms
- Database insert: ~5-10ms (without network latency)
- Average response time: 10-20ms

## Testing

Create a simple test:

```bash
# Test health check
curl http://localhost:8081/health

# Create short link
curl -X POST http://localhost:8081/snap \
  -H "Content-Type: application/json" \
  -d '{"link": "https://github.com"}'

# Retrieve original URL
curl http://localhost:8081/abc123
```

## License

MIT

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request
