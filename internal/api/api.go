package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"csgo-trader/internal/models"
	steamService "csgo-trader/internal/services/steam"
	buffService "csgo-trader/internal/services/buff"
	youpinService "csgo-trader/internal/services/youpin"
	tradingService "csgo-trader/internal/services/trading"
	priceService "csgo-trader/internal/services/price"
	"csgo-trader/internal/websocket"
)

type APIHandler struct {
	db            *gorm.DB
	steamService  *steamService.SteamService
	buffService   *buffService.BuffService
	youpinService *youpinService.YoupinService
	tradingService *tradingService.TradingService
	priceService  *priceService.PriceService
	wsHub         *websocket.Hub
}

func SetupRoutes(r *gin.RouterGroup, db *gorm.DB, steam *steamService.SteamService, buff *buffService.BuffService, youpin *youpinService.YoupinService, trading *tradingService.TradingService, price *priceService.PriceService, wsHub *websocket.Hub) {
	handler := &APIHandler{
		db:            db,
		steamService:  steam,
		buffService:   buff,
		youpinService: youpin,
		tradingService: trading,
		priceService:  price,
		wsHub:         wsHub,
	}

	// Auth routes
	auth := r.Group("/auth")
	{
		auth.GET("/steam/login", handler.SteamLogin)
		auth.GET("/steam/callback", handler.SteamCallback)
		auth.POST("/logout", handler.Logout)
		auth.GET("/me", handler.GetCurrentUser)
	}

	// Market routes
	market := r.Group("/market")
	{
		market.GET("/items", handler.GetMarketItems)
		market.GET("/items/:id", handler.GetItem)
		market.GET("/items/:id/prices", handler.GetItemPrices)
		market.GET("/items/:id/chart", handler.GetPriceChart)
		market.GET("/items/:id/trend", handler.GetItemTrend)
		market.GET("/arbitrage", handler.GetArbitrageOpportunities)
		market.GET("/movers", handler.GetTopMovers)
	}

	// Trading routes
	trading_routes := r.Group("/trading")
	{
		trading_routes.GET("/strategies", handler.GetStrategies)
		trading_routes.POST("/strategies", handler.CreateStrategy)
		trading_routes.PUT("/strategies/:id", handler.UpdateStrategy)
		trading_routes.DELETE("/strategies/:id", handler.DeleteStrategy)
		trading_routes.POST("/strategies/:id/execute", handler.ExecuteStrategy)
		trading_routes.GET("/trades", handler.GetTrades)
		trading_routes.POST("/buy", handler.BuyItem)
		trading_routes.POST("/sell", handler.SellItem)
	}

	// Inventory routes
	inventory := r.Group("/inventory")
	{
		inventory.GET("/steam/:steamid", handler.GetSteamInventory)
		inventory.GET("/buff/:userid", handler.GetBuffInventory)
		inventory.GET("/youpin/:userid", handler.GetYoupinInventory)
	}

	// Analytics routes
	analytics := r.Group("/analytics")
	{
		analytics.GET("/dashboard", handler.GetDashboard)
		analytics.GET("/performance", handler.GetPerformance)
	}
}

// Auth handlers
func (h *APIHandler) SteamLogin(c *gin.Context) {
	returnURL := c.Query("return_url")
	if returnURL == "" {
		returnURL = "http://localhost:8080/api/v1/auth/steam/callback"
	}
	
	loginURL := h.steamService.GetOpenIDLoginURL(returnURL)
	c.JSON(http.StatusOK, gin.H{"login_url": loginURL})
}

func (h *APIHandler) SteamCallback(c *gin.Context) {
	// Get all query parameters
	params := c.Request.URL.Query()
	
	steamID, err := h.steamService.VerifyOpenIDResponse(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Steam login"})
		return
	}
	
	// Get user info from Steam
	userInfo, err := h.steamService.GetUserInfo(steamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}
	
	// Save or update user in database
	var user models.User
	result := h.db.Where("steam_id = ?", steamID).First(&user)
	if result.Error != nil {
		// Create new user
		user = *userInfo
		if err := h.db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
	} else {
		// Update existing user
		user.Username = userInfo.Username
		user.Avatar = userInfo.Avatar
		h.db.Save(&user)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"user": user,
		"message": "Login successful",
	})
}

func (h *APIHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *APIHandler) GetCurrentUser(c *gin.Context) {
	// This would typically check JWT token
	c.JSON(http.StatusOK, gin.H{"user": nil})
}

// Market handlers
func (h *APIHandler) GetMarketItems(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	platform := c.DefaultQuery("platform", "steam")
	
	var items []models.Item
	offset := (page - 1) * limit
	
	query := h.db.Offset(offset).Limit(limit)
	if search := c.Query("search"); search != "" {
		query = query.Where("market_name LIKE ?", "%"+search+"%")
	}
	
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"page":  page,
		"limit": limit,
	})
}

func (h *APIHandler) GetItem(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	
	var item models.Item
	if err := h.db.First(&item, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"item": item})
}

func (h *APIHandler) GetItemPrices(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	
	prices, err := h.priceService.GetLatestPrices(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"prices": prices})
}

func (h *APIHandler) GetPriceChart(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	
	chart, err := h.priceService.GetPriceChart(uint(id), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"chart": chart})
}

func (h *APIHandler) GetItemTrend(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	platform := c.DefaultQuery("platform", "steam")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	
	trend, err := h.priceService.CalculateTrend(uint(id), platform, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"trend": trend})
}

func (h *APIHandler) GetArbitrageOpportunities(c *gin.Context) {
	minProfit, _ := strconv.ParseFloat(c.DefaultQuery("min_profit", "10"), 64)
	
	opportunities, err := h.priceService.GetArbitrageOpportunities(minProfit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"opportunities": opportunities})
}

func (h *APIHandler) GetTopMovers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	
	movers, err := h.priceService.GetTopMovers(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"movers": movers})
}

// Trading handlers
func (h *APIHandler) GetStrategies(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 32)
	
	var strategies []models.Strategy
	query := h.db.Preload("Item")
	if userID > 0 {
		query = query.Where("user_id = ?", uint(userID))
	}
	
	if err := query.Find(&strategies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"strategies": strategies})
}

func (h *APIHandler) CreateStrategy(c *gin.Context) {
	var strategy models.Strategy
	if err := c.ShouldBindJSON(&strategy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := h.tradingService.CreateStrategy(&strategy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"strategy": strategy})
}

func (h *APIHandler) UpdateStrategy(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := h.tradingService.UpdateStrategy(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Strategy updated successfully"})
}

func (h *APIHandler) DeleteStrategy(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	
	if err := h.tradingService.DeleteStrategy(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Strategy deleted successfully"})
}

func (h *APIHandler) ExecuteStrategy(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	
	if err := h.tradingService.ExecuteStrategy(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Strategy executed successfully"})
}

func (h *APIHandler) GetTrades(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 32)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	
	trades, err := h.tradingService.GetUserTrades(uint(userID), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"trades": trades})
}

func (h *APIHandler) BuyItem(c *gin.Context) {
	var request struct {
		ItemID   uint    `json:"item_id"`
		Platform string  `json:"platform"`
		Price    float64 `json:"price"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Implementation would depend on the platform
	c.JSON(http.StatusOK, gin.H{"message": "Buy order placed"})
}

func (h *APIHandler) SellItem(c *gin.Context) {
	var request struct {
		AssetID  string  `json:"asset_id"`
		Platform string  `json:"platform"`
		Price    float64 `json:"price"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Implementation would depend on the platform
	c.JSON(http.StatusOK, gin.H{"message": "Sell order placed"})
}

// Inventory handlers
func (h *APIHandler) GetSteamInventory(c *gin.Context) {
	steamID := c.Param("steamid")
	
	inventory, err := h.steamService.GetUserInventory(steamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"inventory": inventory})
}

func (h *APIHandler) GetBuffInventory(c *gin.Context) {
	userID := c.Param("userid")
	
	inventory, err := h.buffService.GetUserInventory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"inventory": inventory})
}

func (h *APIHandler) GetYoupinInventory(c *gin.Context) {
	userID := c.Param("userid")
	
	inventory, err := h.youpinService.GetUserInventory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"inventory": inventory})
}

// Analytics handlers
func (h *APIHandler) GetDashboard(c *gin.Context) {
	// Get recent trades
	trades, _ := h.tradingService.GetUserTrades(1, 10)
	
	// Get arbitrage opportunities
	opportunities, _ := h.priceService.GetArbitrageOpportunities(10)
	
	// Get top movers
	movers, _ := h.priceService.GetTopMovers(5)
	
	c.JSON(http.StatusOK, gin.H{
		"recent_trades":   trades,
		"opportunities":   opportunities,
		"top_movers":      movers,
		"timestamp":       time.Now(),
	})
}

func (h *APIHandler) GetPerformance(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 32)
	
	// Calculate performance metrics
	var totalProfit float64
	var totalTrades int64
	
	h.db.Model(&models.Trade{}).
		Where("user_id = ? AND status = ?", uint(userID), "completed").
		Count(&totalTrades)
	
	c.JSON(http.StatusOK, gin.H{
		"total_profit": totalProfit,
		"total_trades": totalTrades,
		"success_rate": 0.85, // This would be calculated
		"roi":          0.15,  // This would be calculated
	})
}