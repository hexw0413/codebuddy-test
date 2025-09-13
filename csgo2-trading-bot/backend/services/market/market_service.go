package market

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"csgo2-trading-bot/models"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Service struct {
	db    *gorm.DB
	redis *redis.Client
	ctx   context.Context
}

func NewService(db *gorm.DB, redis *redis.Client) *Service {
	return &Service{
		db:    db,
		redis: redis,
		ctx:   context.Background(),
	}
}

// GetMarketItems 获取市场物品列表
func (s *Service) GetMarketItems(page, pageSize int, filters map[string]interface{}) ([]models.Item, int64, error) {
	var items []models.Item
	var total int64

	query := s.db.Model(&models.Item{})

	// 应用过滤器
	if itemType, ok := filters["type"].(string); ok && itemType != "" {
		query = query.Where("type = ?", itemType)
	}
	if rarity, ok := filters["rarity"].(string); ok && rarity != "" {
		query = query.Where("rarity = ?", rarity)
	}
	if minPrice, ok := filters["min_price"].(float64); ok {
		query = query.Where("current_price >= ?", minPrice)
	}
	if maxPrice, ok := filters["max_price"].(float64); ok {
		query = query.Where("current_price <= ?", maxPrice)
	}

	// 获取总数
	query.Count(&total)

	// 分页
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&items).Error

	return items, total, err
}

// GetItemDetails 获取物品详情
func (s *Service) GetItemDetails(itemID uint) (*models.Item, error) {
	var item models.Item
	if err := s.db.First(&item, itemID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// GetPriceHistory 获取价格历史
func (s *Service) GetPriceHistory(itemID uint, days int) ([]models.PriceHistory, error) {
	var history []models.PriceHistory
	
	startDate := time.Now().AddDate(0, 0, -days)
	
	err := s.db.Where("item_id = ? AND recorded_at >= ?", itemID, startDate).
		Order("recorded_at ASC").
		Find(&history).Error

	return history, err
}

// GetMarketTrends 获取市场趋势
func (s *Service) GetMarketTrends() (map[string]interface{}, error) {
	trends := make(map[string]interface{})

	// 获取热门物品
	var hotItems []models.Item
	s.db.Order("volume_24h DESC").Limit(10).Find(&hotItems)
	trends["hot_items"] = hotItems

	// 获取价格上涨最多的物品
	var risingItems []models.Item
	s.db.Raw(`
		SELECT i.*, 
		       ((i.current_price - i.avg_price_7days) / i.avg_price_7days * 100) as price_change
		FROM items i
		WHERE i.avg_price_7days > 0
		ORDER BY price_change DESC
		LIMIT 10
	`).Scan(&risingItems)
	trends["rising_items"] = risingItems

	// 获取价格下跌最多的物品
	var fallingItems []models.Item
	s.db.Raw(`
		SELECT i.*, 
		       ((i.current_price - i.avg_price_7days) / i.avg_price_7days * 100) as price_change
		FROM items i
		WHERE i.avg_price_7days > 0
		ORDER BY price_change ASC
		LIMIT 10
	`).Scan(&fallingItems)
	trends["falling_items"] = fallingItems

	// 获取市场总览
	var marketOverview struct {
		TotalItems      int64   `json:"total_items"`
		TotalVolume24h  int64   `json:"total_volume_24h"`
		AvgPrice        float64 `json:"avg_price"`
		MedianPrice     float64 `json:"median_price"`
	}
	
	s.db.Model(&models.Item{}).Count(&marketOverview.TotalItems)
	s.db.Model(&models.Item{}).Select("SUM(volume_24h) as total_volume_24h, AVG(current_price) as avg_price").Scan(&marketOverview)
	
	// 计算中位数价格
	var prices []float64
	s.db.Model(&models.Item{}).Pluck("current_price", &prices)
	if len(prices) > 0 {
		marketOverview.MedianPrice = calculateMedian(prices)
	}
	
	trends["overview"] = marketOverview

	return trends, nil
}

// UpdateItemPrice 更新物品价格
func (s *Service) UpdateItemPrice(itemID uint, price float64, platform string) error {
	// 更新物品当前价格
	if err := s.db.Model(&models.Item{}).Where("id = ?", itemID).
		Updates(map[string]interface{}{
			"current_price": price,
			"last_updated":  time.Now(),
		}).Error; err != nil {
		return err
	}

	// 记录价格历史
	priceHistory := models.PriceHistory{
		ItemID:     itemID,
		Price:      price,
		Platform:   platform,
		RecordedAt: time.Now(),
	}
	
	if err := s.db.Create(&priceHistory).Error; err != nil {
		return err
	}

	// 更新Redis缓存
	cacheKey := fmt.Sprintf("item:price:%d", itemID)
	priceData, _ := json.Marshal(map[string]interface{}{
		"price":    price,
		"platform": platform,
		"updated":  time.Now(),
	})
	s.redis.Set(s.ctx, cacheKey, priceData, 5*time.Minute)

	return nil
}

// GetRealtimePrice 获取实时价格（优先从缓存）
func (s *Service) GetRealtimePrice(itemID uint) (float64, error) {
	// 先尝试从Redis获取
	cacheKey := fmt.Sprintf("item:price:%d", itemID)
	data, err := s.redis.Get(s.ctx, cacheKey).Result()
	
	if err == nil {
		var priceData map[string]interface{}
		if err := json.Unmarshal([]byte(data), &priceData); err == nil {
			if price, ok := priceData["price"].(float64); ok {
				return price, nil
			}
		}
	}

	// 从数据库获取
	var item models.Item
	if err := s.db.Select("current_price").First(&item, itemID).Error; err != nil {
		return 0, err
	}

	return item.CurrentPrice, nil
}

// SubscribePriceUpdates 订阅价格更新
func (s *Service) SubscribePriceUpdates(itemIDs []uint) (<-chan PriceUpdate, error) {
	updates := make(chan PriceUpdate, 100)
	
	// 使用Redis发布订阅
	pubsub := s.redis.Subscribe(s.ctx, generatePriceChannels(itemIDs)...)
	
	go func() {
		defer close(updates)
		for msg := range pubsub.Channel() {
			var update PriceUpdate
			if err := json.Unmarshal([]byte(msg.Payload), &update); err == nil {
				updates <- update
			}
		}
	}()

	return updates, nil
}

// PriceUpdate 价格更新结构
type PriceUpdate struct {
	ItemID   uint      `json:"item_id"`
	Price    float64   `json:"price"`
	Platform string    `json:"platform"`
	Time     time.Time `json:"time"`
}

// RecordMarketSnapshot 记录市场快照
func (s *Service) RecordMarketSnapshot(itemID uint, platform string, data models.MarketData) error {
	data.ItemID = itemID
	data.Platform = platform
	data.SnapshotTime = time.Now()
	
	return s.db.Create(&data).Error
}

// GetMarketAnalysis 获取市场分析
func (s *Service) GetMarketAnalysis(itemID uint) (map[string]interface{}, error) {
	analysis := make(map[string]interface{})
	
	// 获取最近30天的价格数据
	var priceHistory []models.PriceHistory
	startDate := time.Now().AddDate(0, 0, -30)
	s.db.Where("item_id = ? AND recorded_at >= ?", itemID, startDate).
		Order("recorded_at ASC").
		Find(&priceHistory)
	
	if len(priceHistory) == 0 {
		return analysis, nil
	}
	
	// 计算统计指标
	prices := make([]float64, len(priceHistory))
	for i, h := range priceHistory {
		prices[i] = h.Price
	}
	
	analysis["min_price"] = findMin(prices)
	analysis["max_price"] = findMax(prices)
	analysis["avg_price"] = calculateAverage(prices)
	analysis["std_dev"] = calculateStdDev(prices)
	analysis["volatility"] = calculateVolatility(prices)
	
	// 计算移动平均
	analysis["ma_7"] = calculateMA(prices, 7)
	analysis["ma_14"] = calculateMA(prices, 14)
	analysis["ma_30"] = calculateMA(prices, 30)
	
	// 计算RSI
	analysis["rsi"] = calculateRSI(prices, 14)
	
	// 趋势判断
	trend := "neutral"
	if len(prices) >= 7 {
		recent := prices[len(prices)-7:]
		if isUptrend(recent) {
			trend = "bullish"
		} else if isDowntrend(recent) {
			trend = "bearish"
		}
	}
	analysis["trend"] = trend
	
	return analysis, nil
}

// 辅助函数
func calculateMedian(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	if len(prices)%2 == 0 {
		return (prices[len(prices)/2-1] + prices[len(prices)/2]) / 2
	}
	return prices[len(prices)/2]
}

func findMin(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	min := prices[0]
	for _, p := range prices {
		if p < min {
			min = p
		}
	}
	return min
}

func findMax(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	max := prices[0]
	for _, p := range prices {
		if p > max {
			max = p
		}
	}
	return max
}

func calculateAverage(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	sum := 0.0
	for _, p := range prices {
		sum += p
	}
	return sum / float64(len(prices))
}

func calculateStdDev(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	avg := calculateAverage(prices)
	sum := 0.0
	for _, p := range prices {
		sum += (p - avg) * (p - avg)
	}
	return sum / float64(len(prices))
}

func calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}
	return calculateStdDev(returns)
}

func calculateMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

func calculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50 // 中性值
	}
	
	gains := 0.0
	losses := 0.0
	
	for i := len(prices) - period; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}
	
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	
	if avgLoss == 0 {
		return 100
	}
	
	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))
	
	return rsi
}

func isUptrend(prices []float64) bool {
	if len(prices) < 2 {
		return false
	}
	upCount := 0
	for i := 1; i < len(prices); i++ {
		if prices[i] > prices[i-1] {
			upCount++
		}
	}
	return upCount > len(prices)/2
}

func isDowntrend(prices []float64) bool {
	if len(prices) < 2 {
		return false
	}
	downCount := 0
	for i := 1; i < len(prices); i++ {
		if prices[i] < prices[i-1] {
			downCount++
		}
	}
	return downCount > len(prices)/2
}

func generatePriceChannels(itemIDs []uint) []string {
	channels := make([]string, len(itemIDs))
	for i, id := range itemIDs {
		channels[i] = fmt.Sprintf("price:update:%d", id)
	}
	return channels
}