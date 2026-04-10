#!/bin/bash
cd "$(dirname "$0")"

# Kill existing process on port 8022
OLD_PID=$(lsof -ti:8022 2>/dev/null)
if [ -n "$OLD_PID" ]; then
  echo ">>> Killing old process on port 8022 (PID: $OLD_PID)..."
  kill -9 $OLD_PID 2>/dev/null
  sleep 1
fi

echo "========================================"
echo "  EasyLLM Server"
echo "  Web UI: http://localhost:8022"
echo "  API:    http://localhost:8022/api/v1"
echo "========================================"
echo ""
echo ">>> Starting... (Ctrl+C to stop)"
echo ""

go run main.go "$@"
