# Bamboo Mapper Backend API

Backend API for Bamboo Mapper - a bamboo mapping application that tracks bamboo locations via map pinpoints.

## Tech Stack

![Go](https://img.shields.io/badge/Go-1.25.5-00ADD8?style=for-the-badge&logo=go&logoColor=white)

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL
- Google Cloud service account (for image uploads)

### Installation

```bash
# Install dependencies
go mod download

# Install dev tools
make deps

# Set up environment variables (copy from .env.example)
cp .env.example .env

# Run migrations
make migrate-up

# Start the server
make run
```

### Available Commands

```bash
make run          # Run the API server
make build        # Build binary to bin/api
make test         # Run all tests
make sqlc         # Generate repository code from SQL queries
make migrate-up   # Run database migrations
make migrate-down # Rollback migrations
make deps         # Install dev tools
make tidy         # Clean up go.mod
```

---

## API Documentation

### Base URL

```
/api/v1
```

### Response Format

All responses follow this structure:

```json
{
  "meta": {
    "success": true,
    "message": "Operation successful"
  },
  "data": {}
}
```

Error responses include details when applicable:

```json
{
  "meta": {
    "success": false,
    "message": "Validation failed",
    "details": {
      "email": "Invalid email format"
    }
  },
  "data": null
}
```

### Authentication

Protected endpoints require JWT Bearer token:

```
Authorization: Bearer {access_token}
```

- Access token expires in 1 hour
- Use refresh token to obtain new access token

---

## Endpoints

### Health Check

| Method | Endpoint  | Auth | Description         |
|--------|-----------|------|---------------------|
| GET    | `/health` | No   | API health check    |

**Response:** `200 OK` with body `"OK"`

---

### Authentication

| Method | Endpoint              | Auth | Description              |
|--------|-----------------------|------|--------------------------|
| POST   | `/api/v1/auth/register` | No   | Register new user        |
| POST   | `/api/v1/auth/login`    | No   | Login and get tokens     |
| POST   | `/api/v1/auth/refresh`  | No   | Refresh access token     |
| GET    | `/api/v1/auth/me`       | Yes  | Get current user profile |
| POST   | `/api/v1/auth/logout`   | Yes  | Logout and invalidate token |

#### POST `/api/v1/auth/register`

Register a new user account.

**Request Body:**
```json
{
  "email": "user@example.com",
  "name": "John Doe",
  "password": "securepassword123"
}
```

| Field    | Type   | Required | Validation            |
|----------|--------|----------|-----------------------|
| email    | string | Yes      | Valid email format    |
| name     | string | Yes      | Max 100 characters    |
| password | string | Yes      | Min 8 characters      |

**Response (201 Created):**
```json
{
  "meta": {
    "success": true,
    "message": "User registered successfully"
  },
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "name": "John Doe",
    "role": "user",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

**Errors:**
- `400` - Validation failed
- `409` - Email already registered

---

#### POST `/api/v1/auth/login`

Authenticate and receive access tokens.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response (200 OK):**
```json
{
  "meta": {
    "success": true,
    "message": "Login successful"
  },
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "dGhpcyBpcyBhIHJlZnJl...",
    "token_type": "Bearer",
    "expires_in": 3600,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "name": "John Doe",
      "role": "user",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  }
}
```

**Errors:**
- `400` - Validation failed
- `401` - Invalid email or password

---

#### POST `/api/v1/auth/refresh`

Get new access token using refresh token.

**Request Body:**
```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJl..."
}
```

**Response (200 OK):**
```json
{
  "meta": {
    "success": true,
    "message": "Token refreshed successfully"
  },
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "bmV3IHJlZnJlc2ggdG9r...",
    "token_type": "Bearer",
    "expires_in": 3600
  }
}
```

**Errors:**
- `400` - Validation failed
- `401` - Invalid or expired refresh token

---

#### GET `/api/v1/auth/me`

Get current authenticated user's profile.

**Headers:**
```
Authorization: Bearer {access_token}
```

**Response (200 OK):**
```json
{
  "meta": {
    "success": true,
    "message": "User retrieved successfully"
  },
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "name": "John Doe",
    "role": "user",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

**Errors:**
- `401` - Unauthorized

---

#### POST `/api/v1/auth/logout`

Logout and invalidate current token.

**Headers:**
```
Authorization: Bearer {access_token}
```

**Response (200 OK):**
```json
{
  "meta": {
    "success": true,
    "message": "Logged out successfully"
  },
  "data": null
}
```

---

### Markers

| Method | Endpoint                      | Auth | Description                     |
|--------|-------------------------------|------|---------------------------------|
| GET    | `/api/v1/markers/`            | Yes  | List all markers (lightweight)  |
| GET    | `/api/v1/markers/{id}`        | Yes  | Get marker by ID (full details) |
| GET    | `/api/v1/markers/code/{code}` | No   | Get marker by short code (QR)   |
| POST   | `/api/v1/markers/`            | Yes  | Create new marker               |
| PUT    | `/api/v1/markers/{id}`        | Yes  | Update marker                   |
| DELETE | `/api/v1/markers/{id}`        | Yes  | Delete marker                   |
| GET    | `/api/v1/markers/{id}/qr`     | Yes  | Get QR code image               |

---

#### GET `/api/v1/markers/`

Get all markers (lightweight response for map display).

**Headers:**
```
Authorization: Bearer {access_token}
```

**Response (200 OK):**
```json
{
  "meta": {
    "success": true,
    "message": "Markers retrieved successfully"
  },
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "short_code": "ABC123",
      "name": "Bamboo Cluster A",
      "latitude": "-7.797068",
      "longitude": "110.370529"
    }
  ]
}
```

---

#### GET `/api/v1/markers/{id}`

Get full marker details by UUID.

**Headers:**
```
Authorization: Bearer {access_token}
```

**Response (200 OK):**
```json
{
  "meta": {
    "success": true,
    "message": "Marker retrieved successfully"
  },
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "short_code": "ABC123",
    "creator_id": "660e8400-e29b-41d4-a716-446655440000",
    "name": "Bamboo Cluster A",
    "description": "Large bamboo cluster near the river",
    "strain": "Bambusa vulgaris",
    "quantity": 50,
    "latitude": "-7.797068",
    "longitude": "110.370529",
    "image_url": "https://drive.google.com/...",
    "owner_name": "Pak Bambang",
    "owner_contact": "081234567890",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

**Errors:**
- `400` - Invalid marker ID format
- `404` - Marker not found

---

#### GET `/api/v1/markers/code/{shortCode}`

Get marker by short code (used for QR code scanning, no auth required).

**Response (200 OK):**
Same as GET `/api/v1/markers/{id}`

**Errors:**
- `404` - Marker not found

---

#### POST `/api/v1/markers/`

Create a new bamboo marker.

**Headers:**
```
Authorization: Bearer {access_token}
Content-Type: multipart/form-data
```

**Form Data:**

| Field         | Type    | Required | Description                    |
|---------------|---------|----------|--------------------------------|
| name          | string  | Yes      | Marker name                    |
| latitude      | string  | Yes      | GPS latitude                   |
| longitude     | string  | Yes      | GPS longitude                  |
| description   | string  | No       | Detailed description           |
| strain        | string  | No       | Bamboo species/strain          |
| quantity      | integer | No       | Number of bamboo (non-negative)|
| owner_name    | string  | No       | Land owner's name              |
| owner_contact | string  | No       | Land owner's contact           |
| image         | file    | No       | Image file (max 10MB)          |

**Response (201 Created):**
```json
{
  "meta": {
    "success": true,
    "message": "Marker created successfully"
  },
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "short_code": "ABC123",
    "creator_id": "660e8400-e29b-41d4-a716-446655440000",
    "name": "Bamboo Cluster A",
    "description": "Large bamboo cluster near the river",
    "strain": "Bambusa vulgaris",
    "quantity": 50,
    "latitude": "-7.797068",
    "longitude": "110.370529",
    "image_url": "https://drive.google.com/...",
    "owner_name": "Pak Bambang",
    "owner_contact": "081234567890",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

**Errors:**
- `400` - Validation failed
- `401` - Unauthorized

---

#### PUT `/api/v1/markers/{id}`

Update an existing marker. Only provided fields are updated.

**Headers:**
```
Authorization: Bearer {access_token}
Content-Type: multipart/form-data
```

**Form Data:** Same as POST (all fields optional)

**Response (200 OK):**
Same structure as POST response with updated data.

**Errors:**
- `400` - Invalid marker ID / Validation failed
- `404` - Marker not found

---

#### DELETE `/api/v1/markers/{id}`

Delete a marker.

**Headers:**
```
Authorization: Bearer {access_token}
```

**Response (200 OK):**
```json
{
  "meta": {
    "success": true,
    "message": "Marker deleted successfully"
  },
  "data": null
}
```

**Errors:**
- `400` - Invalid marker ID format
- `404` - Marker not found

---

#### GET `/api/v1/markers/{id}/qr`

Generate and download QR code image for a marker.

**Headers:**
```
Authorization: Bearer {access_token}
```

**Response (200 OK):**
- Content-Type: `image/png`
- Content-Disposition: `inline; filename="{shortCode}.png"`
- Body: PNG image file

The QR code encodes a deep link URL: `{DEEP_LINK_BASE_URL}/marker/{shortCode}`

**Errors:**
- `400` - Invalid marker ID format
- `404` - Marker not found

---

## Environment Variables

| Variable              | Description                          | Required |
|-----------------------|--------------------------------------|----------|
| `ENVIRONMENT`         | Environment (development/production) | Yes      |
| `PORT`                | Server port                          | Yes      |
| `DATABASE_URL`        | PostgreSQL connection string         | Yes      |
| `JWT_SECRET`          | Secret key for JWT signing           | Yes      |
| `GOOGLE_CREDENTIALS`  | Google Cloud service account JSON    | Yes      |
| `GOOGLE_DRIVE_FOLDER` | Google Drive folder ID for uploads   | Yes      |
| `DEEP_LINK_BASE_URL`  | Base URL for QR code deep links      | Yes      |

---

## Project Structure

```
bamboo-mapper-backend/
├── cmd/api/
│   └── main.go              # Entry point, router setup
├── internal/
│   ├── config/              # Environment configuration
│   ├── database/            # Database connection
│   ├── handler/             # HTTP handlers
│   ├── middleware/          # Auth middleware
│   ├── model/               # Domain models
│   ├── repository/          # sqlc-generated DB layer
│   │   └── queries/         # SQL query files
│   └── service/             # Business logic
├── migrations/              # Database migrations
├── Makefile
└── README.md
```

---

**KKN-PPM UGM Sapuran Berperan 2025**
