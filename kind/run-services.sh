#!/bin/bash

# Script to run both services
# Usage: ./run-services.sh

echo "Starting Message Service on port 8080..."
cd services/message
PORT=8080 NOTIFICATION_SERVICE_URL=http://localhost:8081 go run main.go &
MESSAGE_PID=$!

echo "Starting Real-time Notification Service on port 8081..."
cd ../real-time-ntfn
PORT=8081 go run main.go &
NOTIFICATION_PID=$!

echo ""
echo "Services started!"
echo "Message Service PID: $MESSAGE_PID"
echo "Real-time Notification Service PID: $NOTIFICATION_PID"
echo ""
echo "Press Ctrl+C to stop all services"

# Wait for user interrupt
trap "kill $MESSAGE_PID $NOTIFICATION_PID; exit" INT TERM

wait

