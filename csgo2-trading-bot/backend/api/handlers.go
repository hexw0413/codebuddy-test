package api

import (
	"net/http"
	"strconv"

	"csgo2-trading-bot/models"
	"csgo2-trading-bot/services/auth"
	"csgo2-trading-bot/services/market"
	"csgo2-trading-bot/services/trading"

	"github.com/gin-gonic/gin"
)

// Auth Handlers

func SteamLogin(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		loginURL := authService.GenerateSteamLoginURL()
		c.JSON(http.StatusOK, gin.H{
			"login_url": loginURL,
		})
	}
}

func SteamCallback(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取OpenID参数
		query := c.Request.URL.Query()
		
		// 验证Steam登录
		user, err := authService.VerifySteamLogin(query)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// 生成JWT
		token, err := authService.GenerateJWT(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user":  user,
		})
	}
}

func VerifyToken(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SharedSecret string `json:"shared_secret"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 生成TOTP令牌
		token, err := authService.GenerateTOTP(req.SharedSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
		})
	}
}

func Logout(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 清除会话（如果使用）
		c.JSON(http.StatusOK, gin.H{
			"message": "logged out successfully",
		})
	}
}

// Market Handlers

func GetMarketItems(marketService *market.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
		
		filters := make(map[string]interface{})
		if itemType := c.Query("type"); itemType != "" {
			filters["type"] = itemType
		}
		if rarity := c.Query("rarity"); rarity != "" {
			filters["rarity"] = rarity
		}
		if minPrice := c.Query("min_price"); minPrice != "" {
			if price, err := strconv.ParseFloat(minPrice, 64); err == nil {
				filters["min_price"] = price
			}
		}
		if maxPrice := c.Query("max_price"); maxPrice != "" {
			if price, err := strconv.ParseFloat(maxPrice, 64); err == nil {
				filters["max_price"] = price
			}
		}

		items, total, err := marketService.GetMarketItems(page, pageSize, filters)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items": items,
			"total": total,
			"page":  page,
			"page_size": pageSize,
		})
	}
}

func GetItemDetails(marketService *market.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		itemID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
			return
		}

		item, err := marketService.GetItemDetails(uint(itemID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

func GetPriceHistory(marketService *market.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		itemID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
			return
		}

		days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

		history, err := marketService.GetPriceHistory(uint(itemID), days)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"history": history,
			"days":    days,
		})
	}
}

func GetMarketTrends(marketService *market.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		trends, err := marketService.GetMarketTrends()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, trends)
	}
}

// Trading Handlers

func GetInventory(tradingService *trading.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")

		inventory, err := tradingService.GetInventory(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"inventory": inventory,
		})
	}
}

func CreateBuyOrder(tradingService *trading.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")

		var req struct {
			ItemID   uint    `json:"item_id" binding:"required"`
			Price    float64 `json:"price" binding:"required,min=0"`
			Quantity int     `json:"quantity" binding:"required,min=1"`
			Platform string  `json:"platform" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		order, err := tradingService.CreateBuyOrder(userID, req.ItemID, req.Price, req.Quantity, req.Platform)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, order)
	}
}

func CreateSellOrder(tradingService *trading.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")

		var req struct {
			ItemID   uint    `json:"item_id" binding:"required"`
			Price    float64 `json:"price" binding:"required,min=0"`
			Quantity int     `json:"quantity" binding:"required,min=1"`
			Platform string  `json:"platform" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		order, err := tradingService.CreateSellOrder(userID, req.ItemID, req.Price, req.Quantity, req.Platform)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, order)
	}
}

func GetOrders(tradingService *trading.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		status := c.Query("status")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

		orders, total, err := tradingService.GetOrders(userID, status, page, pageSize)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"orders":    orders,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})
	}
}

func CancelOrder(tradingService *trading.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		orderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
			return
		}

		if err := tradingService.CancelOrder(uint(orderID), userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "order cancelled successfully",
		})
	}
}

// Strategy Handlers

func GetStrategies(tradingService *trading.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")

		strategies, err := tradingService.GetStrategies(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"strategies": strategies,
		})
	}
}

func CreateStrategy(tradingService *trading.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")

		var strategy models.Strategy
		if err := c.ShouldBindJSON(&strategy); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := tradingService.CreateStrategy(userID, &strategy); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, strategy)
	}
}

func UpdateStrategy(tradingService *trading.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		strategyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid strategy id"})
			return
		}

		var updates map[string]interface{}
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := tradingService.UpdateStrategy(uint(strategyID), userID, updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "strategy updated successfully",
		})
	}
}

func DeleteStrategy(tradingService *trading.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		strategyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid strategy id"})
			return
		}

		if err := tradingService.DeleteStrategy(uint(strategyID), userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "strategy deleted successfully",
		})
	}
}

func ActivateStrategy(tradingService *trading.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		strategyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid strategy id"})
			return
		}

		if err := tradingService.ActivateStrategy(uint(strategyID), userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "strategy activated successfully",
		})
	}
}

func DeactivateStrategy(tradingService *trading.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		strategyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid strategy id"})
			return
		}

		if err := tradingService.DeactivateStrategy(uint(strategyID), userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "strategy deactivated successfully",
		})
	}
}

// Stats Handlers

func GetProfitStats(tradingService *trading.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		period := c.DefaultQuery("period", "month")

		stats, err := tradingService.GetProfitStats(userID, period)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}

func GetTradingStats(tradingService *trading.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")

		stats, err := tradingService.GetTradingStats(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}