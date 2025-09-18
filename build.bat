@echo off
REM CSGO2 Auto Trading Platform - Windows Build Script
REM This script builds the entire application for Windows deployment

echo ====================================
echo CSGO2 Auto Trading Platform Builder
echo ====================================
echo.

REM Check if Go is installed
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: Go is not installed or not in PATH
    echo Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

REM Check if Node.js is installed
node --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: Node.js is not installed or not in PATH
    echo Please install Node.js from https://nodejs.org/
    pause
    exit /b 1
)

REM Check if Python is installed
python --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: Python is not installed or not in PATH
    echo Please install Python from https://python.org/
    pause
    exit /b 1
)

echo All prerequisites found!
echo.

REM Create build directory
if not exist "build" mkdir build
if not exist "build\logs" mkdir build\logs
if not exist "build\data" mkdir build\data

echo Building Go backend...
echo.

REM Build Go backend
go mod tidy
if %errorlevel% neq 0 (
    echo ERROR: Failed to tidy Go modules
    pause
    exit /b 1
)

go build -o build\csgo-trader.exe .
if %errorlevel% neq 0 (
    echo ERROR: Failed to build Go backend
    pause
    exit /b 1
)

echo Go backend built successfully!
echo.

echo Installing Python dependencies...
echo.

REM Install Python dependencies
pip install -r requirements.txt
if %errorlevel% neq 0 (
    echo ERROR: Failed to install Python dependencies
    pause
    exit /b 1
)

echo Python dependencies installed!
echo.

echo Building React frontend...
echo.

REM Build React frontend
cd web
call npm install
if %errorlevel% neq 0 (
    echo ERROR: Failed to install npm dependencies
    cd ..
    pause
    exit /b 1
)

call npm run build
if %errorlevel% neq 0 (
    echo ERROR: Failed to build React frontend
    cd ..
    pause
    exit /b 1
)

cd ..

REM Copy built frontend to build directory
if exist "web\build" (
    xcopy /E /I /Y "web\build" "build\web\dist"
    echo Frontend built and copied successfully!
) else (
    echo ERROR: Frontend build directory not found
    pause
    exit /b 1
)

echo.

REM Copy Python files
echo Copying Python data collector...
xcopy /E /I /Y "python" "build\python"

REM Copy configuration files
echo Copying configuration files...
copy ".env.example" "build\.env" >nul 2>&1
copy "README.md" "build\" >nul 2>&1

REM Create start scripts
echo Creating start scripts...

REM Create Windows start script
echo @echo off > build\start.bat
echo echo Starting CSGO2 Auto Trading Platform... >> build\start.bat
echo echo. >> build\start.bat
echo start "Data Collector" python python\main.py >> build\start.bat
echo timeout /t 2 >> build\start.bat
echo start "Main Application" csgo-trader.exe >> build\start.bat
echo echo Both services started! >> build\start.bat
echo echo Check the logs for any errors. >> build\start.bat
echo pause >> build\start.bat

REM Create stop script
echo @echo off > build\stop.bat
echo echo Stopping CSGO2 Auto Trading Platform... >> build\stop.bat
echo taskkill /f /im csgo-trader.exe >nul 2>&1 >> build\stop.bat
echo taskkill /f /im python.exe >nul 2>&1 >> build\stop.bat
echo echo Services stopped! >> build\stop.bat
echo pause >> build\stop.bat

REM Create README for build
echo # CSGO2 Auto Trading Platform > build\BUILD_README.md
echo. >> build\BUILD_README.md
echo ## Setup Instructions >> build\BUILD_README.md
echo. >> build\BUILD_README.md
echo 1. Copy the .env file and configure your API keys: >> build\BUILD_README.md
echo    - STEAM_API_KEY: Get from https://steamcommunity.com/dev/apikey >> build\BUILD_README.md
echo    - BUFF_API_KEY: Get from BUFF163 >> build\BUILD_README.md
echo    - YOUPIN_API_KEY: Get from YouPin898 >> build\BUILD_README.md
echo. >> build\BUILD_README.md
echo 2. Run start.bat to start the application >> build\BUILD_README.md
echo. >> build\BUILD_README.md
echo 3. Open http://localhost:8080 in your browser >> build\BUILD_README.md
echo. >> build\BUILD_README.md
echo 4. Use stop.bat to stop all services >> build\BUILD_README.md
echo. >> build\BUILD_README.md
echo ## File Structure >> build\BUILD_README.md
echo. >> build\BUILD_README.md
echo - csgo-trader.exe: Main Go backend server >> build\BUILD_README.md
echo - python/: Data collection service >> build\BUILD_README.md
echo - web/: Frontend files >> build\BUILD_README.md
echo - logs/: Application logs >> build\BUILD_README.md
echo - data/: Database files >> build\BUILD_README.md

echo.
echo ====================================
echo Build completed successfully!
echo ====================================
echo.
echo Build location: %CD%\build
echo.
echo Next steps:
echo 1. Navigate to the build directory
echo 2. Configure your API keys in .env file
echo 3. Run start.bat to start the application
echo 4. Open http://localhost:8080 in your browser
echo.
pause