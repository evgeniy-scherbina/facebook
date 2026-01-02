# Live Chat Backend - Microservices Architecture

A real-time chat application built with Go using a microservices architecture. The application is split into two services: a message service for sending messages via HTTP REST API, and a real-time notification service for receiving notifications via Server-Sent Events (SSE).

## Architecture

The application consists of two independent services:

1. **Message Service** (`services/message/`)
   - HTTP REST API for sending messages
   - Endpoint: `POST /messages`
   - Default port: `8080`
   - Notifies the real-time notification service when messages are received

2. **Real-time Notification Service** (`services/real-time-ntfn/`)
   - SSE (Server-Sent Events) endpoint for real-time notifications
   - Endpoint: `GET /events` (SSE stream)
   - Endpoint: `POST /notify` (internal, used by message service)
   - Default port: `8081`
   - Broadcasts notifications to all connected clients

## Features

- RESTful API for sending messages
- Real-time notifications via SSE
- Automatic reconnection handling
- Simple HTML client for testing
- Microservices architecture with service-to-service communication

## Prerequisites

**For Docker Compose:**
- Docker and Docker Compose installed
- Internet connection (for downloading images)

**For Local Development:**
- Go 1.21 or later
- Internet connection (for downloading dependencies)

## Setup

### Option 1: Using Docker Compose (Recommended)

1. Build and start both services:
```bash
docker-compose up --build
```

2. Access the UI at `http://localhost:8080`

3. To stop the services:
```bash
docker-compose down
```

### Option 2: Running Locally

1. Install dependencies:
```bash
go mod download
```

2. Run both services:

**Terminal 1 - Message Service:**
```bash
cd services/message
go run main.go
```

**Terminal 2 - Real-time Notification Service:**
```bash
cd services/real-time-ntfn
go run main.go
```

Or use environment variables to configure ports:
```bash
# Terminal 1
PORT=8080 NOTIFICATION_SERVICE_URL=http://localhost:8081 go run services/message/main.go

# Terminal 2
PORT=8081 go run services/real-time-ntfn/main.go
```

## Usage

1. Start both services (see Setup above)
2. Open `index.html` in your browser (or serve it via a simple HTTP server)
3. Configure service URLs in the browser console if needed:
   ```javascript
   window.MESSAGE_SERVICE_URL = 'http://localhost:8080';
   window.NOTIFICATION_SERVICE_URL = 'http://localhost:8081';
   ```
4. Type a message and press Enter or click Send
5. All connected clients will receive the message in real-time via SSE

## API Endpoints

### Message Service (Port 8080)

- `POST /messages` - Send a message
  ```json
  {
    "content": "Hello, world!",
    "user": "John Doe" // optional
  }
  ```
  Response:
  ```json
  {
    "id": "20240101120000.123456",
    "content": "Hello, world!",
    "user": "John Doe",
    "timestamp": "2024-01-01T12:00:00Z",
    "status": "sent"
  }
  ```

- `GET /health` - Health check endpoint

### Real-time Notification Service (Port 8081)

- `GET /events` - SSE stream for real-time notifications
  - Query parameter: `client_id` (optional)
  - Returns Server-Sent Events stream

- `POST /notify` - Internal endpoint for receiving notifications (used by message service)
  ```json
  {
    "id": "20240101120000.123456",
    "content": "Hello, world!",
    "user": "John Doe",
    "timestamp": "2024-01-01T12:00:00Z",
    "type": "message"
  }
  ```

- `GET /health` - Health check endpoint

## Docker

The project includes Dockerfiles and docker-compose configuration:

- **docker-compose.yml** - Orchestrates both services
- **services/message/Dockerfile** - Message service container
- **services/real-time-ntfn/Dockerfile** - Real-time notification service container

### Docker Compose Commands

```bash
# Build and start services
docker-compose up --build

# Start services in detached mode
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Rebuild and restart
docker-compose up --build --force-recreate
```

### Building Individual Docker Images

```bash
# Build message service
docker build -f services/message/Dockerfile -t message-service .

# Build real-time notification service
docker build -f services/real-time-ntfn/Dockerfile -t real-time-ntfn-service .
```

## Building

To build binaries for both services:

```bash
# Build message service
cd services/message
go build -o message-service main.go

# Build real-time notification service
cd ../real-time-ntfn
go build -o real-time-ntfn-service main.go
```

Then run:
```bash
./message-service
./real-time-ntfn-service
```

## Environment Variables

### Message Service
- `PORT` - Port to listen on (default: `8080`)
- `NOTIFICATION_SERVICE_URL` - URL of the notification service (default: `http://localhost:8081`)

### Real-time Notification Service
- `PORT` - Port to listen on (default: `8081`)

## Development

The services communicate via HTTP:
- When a message is posted to the message service, it makes an HTTP POST request to the notification service
- The notification service broadcasts the notification to all connected SSE clients

For production deployments, consider:
- Using a message broker (Redis Pub/Sub, RabbitMQ, etc.) for better scalability
- Adding authentication and authorization
- Implementing proper error handling and retry logic
- Adding metrics and monitoring
- Using proper service discovery mechanisms
