# Snap-Link API Documentation

## Overview
RESTful API for creating and managing shortened URLs. Built with Fiber, GORM, and PostgreSQL.

## Base URL
```
http://localhost:8081
```

## Endpoints

### 1. Create Short Link

**POST** `/snap`

Creates a new shortened URL from a provided link.

**Authentication:** Requires `X-API-KEY` header with valid API key (constant-time compare)

**Rate Limiting:** Fixed 5 requests per minute per IP (see `internal/ratelimit`).

#### Request Headers
```
Content-Type: application/json
X-API-KEY: your-api-key
```

#### Request Body
```json
{
  "link": "https://example.com/very/long/url/path?param=value"
}
```

#### Query Parameters
None

#### Response (201 Created)
```json
{
  "link": "abc123"
}
```

#### Error Responses

**400 Bad Request** - Invalid request or URL
```json
{
  "error": "Invalid URL format"
}
```

**401 Unauthorized** - Missing or invalid API key
```json
{
  "error": "Missing or invalid X-API-KEY header"
}
```

**429 Too Many Requests** - Rate limit exceeded
```json
{
  "error": "Rate limit exceeded: 5 requests per minute allowed"
}
```

#### Example
```bash
curl -X POST http://localhost:8081/snap \
  -H "Content-Type: application/json" \
  -H "X-API-KEY: your-api-key" \
  -d '{"link": "https://github.com/golang/go"}'

# Response:
# {"link":"xYz9Qk"}
```

---

### 2. Get Original URL

**GET** `/:shortCode`

Retrieves the original URL and metadata for a given short code.
NOTE: returns JSON by design. If you want classic shortener behavior,
issue a `302` to `original_url` client-side.

**Validation:** `shortCode` must be 4-10 alphanumeric chars, else `400`.

#### Path Parameters
| Parameter | Type | Description |
|-----------|------|-------------|
| shortCode | string | The 6-character short code |

#### Response (200 OK)
```json
{
  "original_url": "https://example.com/very/long/url",
  "short_code": "abc123",
  "created_at": "2024-01-15T10:30:00Z"
}
```

#### Error Responses

**404 Not Found** - Short code doesn't exist
```json
{
  "error": "Short link not found"
}
```

#### Example
```bash
curl http://localhost:8081/abc123

# Response:
# {
#   "original_url":"https://github.com/golang/go",
#   "short_code":"xYz9Qk",
#   "created_at":"2024-01-15T10:30:00Z"
# }
```

---

### 3. Health Check

**GET** `/health`

Checks the health status of the API. Pings Postgres: returns `200 healthy`
or `503 unhealthy` when the DB is unreachable.

#### Response (200 OK)
```json
{
  "status": "healthy"
}
```

#### Example
```bash
curl http://localhost:8081/health

# Response:
# {"status":"healthy"}
```

---

## Request/Response Examples

### Example 1: Creating a Short Link
```bash
# Request
POST /snap HTTP/1.1
Host: localhost:8081
Content-Type: application/json
Content-Length: 47

{"link":"https://www.example.com/products/laptop"}

# Response
HTTP/1.1 201 Created
Content-Type: application/json
Content-Length: 19

{"link":"Kp8mQw"}
```

### Example 2: Retrieving Original URL
```bash
# Request
GET /Kp8mQw HTTP/1.1
Host: localhost:8081

# Response
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 129

{
  "original_url":"https://www.example.com/products/laptop",
  "short_code":"Kp8mQw",
  "created_at":"2024-01-15T14:22:30.123456Z"
}
```

### Example 3: Invalid URL
```bash
# Request
POST /snap HTTP/1.1
Host: localhost:8081
Content-Type: application/json

{"link":"not a valid url"}

# Response
HTTP/1.1 400 Bad Request
Content-Type: application/json

{"error":"Invalid URL format"}
```

---

## Error Handling

| Status Code | Description |
|------------|-------------|
| 200 OK | Request successful |
| 201 Created | Resource created successfully |
| 400 Bad Request | Invalid input or request format |
| 401 Unauthorized | Missing or invalid API key |
| 404 Not Found | Resource not found |
| 429 Too Many Requests | Rate limit exceeded (fixed 5/min) |
| 500 Internal Server Error | Server error |
| 503 Service Unavailable | /health DB unreachable |

## Rate Limiting

Rate limiting is implemented on the POST `/snap` endpoint:
- **Limit:** fixed 5 requests per minute (see `internal/ratelimit`)
- **Scope:** Per IP address, fixed 1-minute window
- **Response:** 429 Too Many Requests with error message

Requests exceeding the limit will receive:
```json
{
  "error": "Rate limit exceeded: 5 requests per minute allowed"
}
```

## Authentication

API key authentication is required for POST `/snap`:
- **Header:** `X-API-KEY`
- **Format:** raw API key string (constant-time compare)
- **Configuration:** Set via `API_KEY` environment variable

Missing or invalid API key returns:
```json
{
  "error": "Missing or invalid X-API-KEY header"
}
```

## Pagination

Not applicable for current endpoints.

## Filtering

Not applicable for current endpoints.

## Sorting

Not applicable for current endpoints.

## Data Types

### Short Code
- Type: `string`
- Length: 6 characters
- Alphabet: `[A-Za-z0-9]` (62 combinations)
- Max combinations: 62^6 ≈ 56 trillion

### Original URL
- Type: `string`
- Max length: Text type (PostgreSQL)
- Required format: Valid URL

### Created At
- Type: `datetime` (ISO 8601)
- Format: `YYYY-MM-DDTHH:MM:SS.ffffffZ`
- Timezone: UTC

## Response Format

All responses are in JSON format with proper HTTP status codes.

### Success Response Structure
```json
{
  "link": "value",
  "original_url": "value",
  "short_code": "value",
  "created_at": "value"
}
```

### Error Response Structure
```json
{
  "error": "Error message describing what went wrong"
}
```

## Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| PORT | 8081 | No | HTTP server port |
| DATABASE_URL | - | Yes | PostgreSQL connection string |
| API_KEY | - | Yes | Required for /snap and /config/* (X-API-KEY header) |

## Best Practices

1. **Always use HTTPS** in production
2. **Validate URLs** before shortening
3. **Handle rate limits** with exponential backoff
5. **Monitor error rates** and latencies
6. **Use connection pooling** for database
7. **Implement proper logging** for debugging

## Changelog

### v1.0.0
- Initial release
- POST /snap endpoint
- GET /:shortCode endpoint
- Health check endpoint
- Collision-resistant short codes
- GORM + PostgreSQL integration
- Fiber web framework
