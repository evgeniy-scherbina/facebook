# Live Chat Backend

A simple real-time chat application built with Go and WebSockets.

## Features

- Real-time message broadcasting to all connected clients
- WebSocket-based communication
- Simple HTML client for testing
- Automatic reconnection handling

## Prerequisites

- Go 1.21 or later
- Internet connection (for downloading dependencies)

## Setup

1. Install dependencies:
```bash
go mod download
```

2. Run the server:
```bash
go run main.go
```

The server will start on `http://localhost:8080`

## Usage

1. Open `http://localhost:8080` in your browser
2. Open the same URL in multiple browser tabs/windows to simulate multiple users
3. Type a message and press Enter or click Send
4. All connected clients will receive the message in real-time

## Architecture

- **Hub**: Manages all connected clients and broadcasts messages
- **Client**: Represents a single WebSocket connection
- **WebSocket Endpoint**: `/ws` - handles WebSocket connections
- **Static Files**: `/` - serves the HTML client

## Building

To build a binary:

```bash
go build -o chat-server main.go
```

Then run:
```bash
./chat-server
```

## Development

The server uses the `gorilla/websocket` library for WebSocket support. The hub pattern is used to manage multiple client connections and broadcast messages efficiently.

