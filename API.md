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

**Authentication:** Requires Neon JWT — `Authorization: Bearer <token>`
(verified against `<NEON_AUTH_BASE_URL>/.well-known/jwks.json`, issuer-only, EdDSA).
Reads (`GET /:code`, `GET /health`) stay public.

**Rate Limiting:** Fixed 5 requests per minute per IP (see `internal/ratelimit`).

#### Request Headers
```
Content-Type: application/json
Authorization: Bearer <neon-jwt>
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

**401 Unauthorized** - Missing/invalid/expired Neon JWT
```json
{
  "error": "Invalid or expired token"
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
  -H "Authorization: Bearer $TOKEN" \
  -d '{"link": "https://github.com/golang/go"}'

# Response:
# {"link":"xYz9Qk"}
```

---

### 2. Follow Short Link

**GET** `/:shortCode`

Issues a `302 Found` redirect to the original URL.

**Validation:** `shortCode` must be 4-10 alphanumeric chars, else `400`.

#### Path Parameters
| Parameter | Type | Description |
|-----------|------|-------------|
| shortCode | string | The 6-character short code |

#### Response (302 Found)
```
HTTP/1.1 302 Found
Location: https://example.com/very/long/url
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
curl -i http://localhost:8081/abc123

# Response:
# HTTP/1.1 302 Found
# Location: https://github.com/golang/go
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

### Example 2: Following a Short Link
```bash
# Request
GET /Kp8mQw HTTP/1.1
Host: localhost:8081

# Response
HTTP/1.1 302 Found
Location: https://www.example.com/products/laptop
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
| 200 OK | Request successful (`GET /health`) |
| 201 Created | Resource created successfully |
| 302 Found | Redirect to original URL (`GET /:shortCode`) |
| 400 Bad Request | Invalid input or request format |
| 401 Unauthorized | Missing/invalid/expired Neon JWT |
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

Neon Managed Better Auth (Better Auth). Frontend signs in via SDK with
`NEON_AUTH_BASE_URL`, sends `Authorization: Bearer <jwt>`.
Go verifies EdDSA signature via `<base>/.well-known/jwks.json`,
issuer = origin of base URL, plus expiry (no audience check).
`sub` claim becomes the user ID (`c.Locals("userID")`).
Required for `POST /snap`; reads stay public.

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

JSON for API responses (`POST /snap`, errors); `GET /:shortCode`
responds with a `302` redirect (`Location` header) instead of a body.

### Success Response Structure
```json
{
  "link": "value"
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
| NEON_AUTH_BASE_URL | - | Yes | Neon Managed Better Auth base URL (per-branch); JWKS + issuer derived |

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
