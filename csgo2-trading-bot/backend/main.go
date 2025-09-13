package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"csgo2-trading-bot/api"
	"csgo2-trading-bot/config"
	"csgo2-trading-bot/database"
	"csgo2-trading-bot/services/auth"
	"csgo2-trading-bot/services/market"
	"csgo2-trading-bot/services/trading"
	"csgo2-trading-bot/websocket"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// 初始化日志
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库
	db, err := database.Initialize(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 初始化Redis
	redisClient := database.InitRedis(cfg.Redis)

	// 初始化服务
	authService := auth.NewService(db, redisClient, cfg.Steam)
	marketService := market.NewService(db, redisClient)
	tradingService := trading.NewService(db, redisClient, cfg.Trading)

	// 设置Gin路由
	router := gin.Default()
	
	// 配置CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// API路由
	apiGroup := router.Group("/api/v1")
	{
		// 认证相关
		apiGroup.POST("/auth/steam/login", api.SteamLogin(authService))
		apiGroup.POST("/auth/steam/callback", api.SteamCallback(authService))
		apiGroup.POST("/auth/steam/verify-token", api.VerifyToken(authService))
		apiGroup.POST("/auth/logout", api.Logout(authService))

		// 需要认证的路由
		protected := apiGroup.Group("/")
		protected.Use(api.AuthMiddleware(authService))
		{
			// 市场数据
			protected.GET("/market/items", api.GetMarketItems(marketService))
			protected.GET("/market/items/:id", api.GetItemDetails(marketService))
			protected.GET("/market/items/:id/history", api.GetPriceHistory(marketService))
			protected.GET("/market/trends", api.GetMarketTrends(marketService))

			// 交易相关
			protected.GET("/trading/inventory", api.GetInventory(tradingService))
			protected.POST("/trading/buy", api.CreateBuyOrder(tradingService))
			protected.POST("/trading/sell", api.CreateSellOrder(tradingService))
			protected.GET("/trading/orders", api.GetOrders(tradingService))
			protected.DELETE("/trading/orders/:id", api.CancelOrder(tradingService))

			// 策略管理
			protected.GET("/strategies", api.GetStrategies(tradingService))
			protected.POST("/strategies", api.CreateStrategy(tradingService))
			protected.PUT("/strategies/:id", api.UpdateStrategy(tradingService))
			protected.DELETE("/strategies/:id", api.DeleteStrategy(tradingService))
			protected.POST("/strategies/:id/activate", api.ActivateStrategy(tradingService))
			protected.POST("/strategies/:id/deactivate", api.DeactivateStrategy(tradingService))

			// 统计数据
			protected.GET("/stats/profit", api.GetProfitStats(tradingService))
			protected.GET("/stats/trading", api.GetTradingStats(tradingService))
		}
	}

	// WebSocket连接
	router.GET("/ws", websocket.HandleWebSocket(marketService))

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// 启动服务器
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	// 优雅关闭
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	logrus.Infof("Server started on port %d", cfg.Server.Port)

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	logrus.Info("Server exited")
}