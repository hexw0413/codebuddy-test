package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"csgo-trader/internal/api"
	"csgo-trader/internal/config"
	"csgo-trader/internal/database"
	"csgo-trader/internal/services"
	"csgo-trader/internal/websocket"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Initialize services
	steamService := services.NewSteamService(cfg.SteamAPIKey)
	buffService := services.NewBuffService(cfg.BuffAPIKey)
	youpinService := services.NewYoupinService(cfg.YoupinAPIKey)
	tradingService := services.NewTradingService(db, steamService, buffService, youpinService)
	priceService := services.NewPriceService(db)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Initialize Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Serve static files
	r.Static("/static", "./web/dist")
	r.StaticFile("/", "./web/dist/index.html")

	// API routes
	apiGroup := r.Group("/api/v1")
	api.SetupRoutes(apiGroup, db, steamService, buffService, youpinService, tradingService, priceService, wsHub)

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		websocket.HandleWebSocket(wsHub, c.Writer, c.Request)
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}