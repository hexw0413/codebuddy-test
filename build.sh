#!/bin/bash

# CSGO2 Auto Trading Platform - Linux/macOS Build Script
# This script builds the entire application for deployment

set -e

echo "===================================="
echo "CSGO2 Auto Trading Platform Builder"
echo "===================================="
echo

# Check prerequisites
echo "Checking prerequisites..."

# Check Go
if ! command -v go &> /dev/null; then
    echo "ERROR: Go is not installed"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

# Check Node.js
if ! command -v node &> /dev/null; then
    echo "ERROR: Node.js is not installed"
    echo "Please install Node.js from https://nodejs.org/"
    exit 1
fi

# Check Python
if ! command -v python3 &> /dev/null; then
    echo "ERROR: Python 3 is not installed"
    echo "Please install Python 3"
    exit 1
fi

echo "All prerequisites found!"
echo

# Create build directory
mkdir -p build/{logs,data}

echo "Building Go backend..."
echo

# Build Go backend
go mod tidy
GOOS=linux GOARCH=amd64 go build -o build/csgo-trader .

echo "Go backend built successfully!"
echo

echo "Installing Python dependencies..."
echo

# Install Python dependencies
pip3 install -r requirements.txt

echo "Python dependencies installed!"
echo

echo "Building React frontend..."
echo

# Build React frontend
cd web
npm install
npm run build
cd ..

# Copy built frontend
cp -r web/build build/web/dist

echo "Frontend built and copied successfully!"
echo

# Copy Python files
echo "Copying Python data collector..."
cp -r python build/

# Copy configuration files
echo "Copying configuration files..."
cp .env.example build/.env
cp README.md build/

# Create start scripts
echo "Creating start scripts..."

# Create Linux start script
cat > build/start.sh << 'EOF'
#!/bin/bash

echo "Starting CSGO2 Auto Trading Platform..."
echo

# Start data collector in background
python3 python/main.py &
PYTHON_PID=$!

# Wait a moment
sleep 2

# Start main application
./csgo-trader &
GO_PID=$!

echo "Both services started!"
echo "Python PID: $PYTHON_PID"
echo "Go PID: $GO_PID"
echo
echo "Access the application at http://localhost:8080"
echo "Press Ctrl+C to stop all services"

# Wait for interrupt
trap 'echo "Stopping services..."; kill $PYTHON_PID $GO_PID; exit' INT
wait
EOF

# Create stop script
cat > build/stop.sh << 'EOF'
#!/bin/bash

echo "Stopping CSGO2 Auto Trading Platform..."

pkill -f csgo-trader
pkill -f "python.*main.py"

echo "Services stopped!"
EOF

# Make scripts executable
chmod +x build/start.sh
chmod +x build/stop.sh
chmod +x build/csgo-trader

# Create README for build
cat > build/BUILD_README.md << 'EOF'
# CSGO2 Auto Trading Platform

## Setup Instructions

1. Copy the .env file and configure your API keys:
   - STEAM_API_KEY: Get from https://steamcommunity.com/dev/apikey
   - BUFF_API_KEY: Get from BUFF163
   - YOUPIN_API_KEY: Get from YouPin898

2. Run start.sh to start the application:
   ```bash
   ./start.sh
   ```

3. Open http://localhost:8080 in your browser

4. Use stop.sh to stop all services:
   ```bash
   ./stop.sh
   ```

## File Structure

- csgo-trader: Main Go backend server
- python/: Data collection service
- web/: Frontend files
- logs/: Application logs
- data/: Database files

## Requirements

- Linux or macOS
- Python 3.7+
- Go 1.19+
- Node.js 16+
EOF

echo
echo "===================================="
echo "Build completed successfully!"
echo "===================================="
echo
echo "Build location: $(pwd)/build"
echo
echo "Next steps:"
echo "1. Navigate to the build directory"
echo "2. Configure your API keys in .env file"
echo "3. Run ./start.sh to start the application"
echo "4. Open http://localhost:8080 in your browser"
echo