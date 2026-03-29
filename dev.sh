#!/bin/bash

# Start frontend dev server in background
echo "Starting frontend dev server..."
cd web
npm install
npm run dev &
FRONTEND_PID=$!
cd ..

# Wait a moment for frontend to start
sleep 3

# Start Go backend
echo "Starting Go backend..."
go run main.go

# Clean up frontend process on exit
trap "kill $FRONTEND_PID 2>/dev/null; exit" INT TERM EXIT
