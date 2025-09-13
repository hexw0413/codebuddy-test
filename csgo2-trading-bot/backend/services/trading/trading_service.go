package trading

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"csgo2-trading-bot/config"
	"csgo2-trading-bot/models"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Service struct {
	db      *gorm.DB
	redis   *redis.Client
	config  config.TradingConfig
	ctx     context.Context
}

func NewService(db *gorm.DB, redis *redis.Client, cfg config.TradingConfig) *Service {
	return &Service{
		db:     db,
		redis:  redis,
		config: cfg,
		ctx:    context.Background(),
	}
}

// GetInventory 获取用户库存
func (s *Service) GetInventory(userID uint) ([]models.Inventory, error) {
	var inventory []models.Inventory
	err := s.db.Preload("Item").Where("user_id = ?", userID).Find(&inventory).Error
	return inventory, err
}

// CreateBuyOrder 创建买入订单
func (s *Service) CreateBuyOrder(userID uint, itemID uint, price float64, quantity int, platform string) (*models.Order, error) {
	// 检查用户余额（这里简化处理，实际需要接入支付系统）
	totalCost := price * float64(quantity)
	if !s.checkUserBalance(userID, totalCost) {
		return nil, errors.New("insufficient balance")
	}

	// 创建订单
	order := models.Order{
		UserID:   userID,
		ItemID:   itemID,
		Type:     "buy",
		Status:   "pending",
		Price:    price,
		Quantity: quantity,
		Platform: platform,
	}

	if err := s.db.Create(&order).Error; err != nil {
		return nil, err
	}

	// 异步执行订单
	go s.executeBuyOrder(&order)

	return &order, nil
}

// CreateSellOrder 创建卖出订单
func (s *Service) CreateSellOrder(userID uint, itemID uint, price float64, quantity int, platform string) (*models.Order, error) {
	// 检查库存
	if !s.checkInventory(userID, itemID, quantity) {
		return nil, errors.New("insufficient inventory")
	}

	// 锁定库存
	if err := s.lockInventory(userID, itemID, quantity); err != nil {
		return nil, err
	}

	// 创建订单
	order := models.Order{
		UserID:   userID,
		ItemID:   itemID,
		Type:     "sell",
		Status:   "pending",
		Price:    price,
		Quantity: quantity,
		Platform: platform,
	}

	if err := s.db.Create(&order).Error; err != nil {
		s.unlockInventory(userID, itemID, quantity)
		return nil, err
	}

	// 异步执行订单
	go s.executeSellOrder(&order)

	return &order, nil
}

// GetOrders 获取用户订单
func (s *Service) GetOrders(userID uint, status string, page, pageSize int) ([]models.Order, int64, error) {
	var orders []models.Order
	var total int64

	query := s.db.Model(&models.Order{}).Where("user_id = ?", userID)
	
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Preload("Item").Offset(offset).Limit(pageSize).
		Order("created_at DESC").Find(&orders).Error

	return orders, total, err
}

// CancelOrder 取消订单
func (s *Service) CancelOrder(orderID uint, userID uint) error {
	var order models.Order
	if err := s.db.First(&order, orderID).Error; err != nil {
		return err
	}

	if order.UserID != userID {
		return errors.New("unauthorized")
	}

	if order.Status != "pending" {
		return errors.New("order cannot be cancelled")
	}

	// 如果是卖单，解锁库存
	if order.Type == "sell" {
		s.unlockInventory(order.UserID, order.ItemID, order.Quantity)
	}

	order.Status = "cancelled"
	return s.db.Save(&order).Error
}

// executeBuyOrder 执行买入订单
func (s *Service) executeBuyOrder(order *models.Order) {
	// 根据平台执行不同的购买逻辑
	var err error
	
	switch order.Platform {
	case "buff":
		if s.config.BuffAPI.Enabled {
			err = s.executeBuff Buy(order)
		}
	case "youpin":
		if s.config.YouPin.Enabled {
			err = s.executeYouPinBuy(order)
		}
	case "steam":
		err = s.executeSteamBuy(order)
	default:
		err = errors.New("unsupported platform")
	}

	if err != nil {
		order.Status = "failed"
		order.FailedReason = err.Error()
	} else {
		order.Status = "completed"
		now := time.Now()
		order.ExecutedAt = &now
		
		// 添加到库存
		s.addToInventory(order)
		
		// 记录交易
		s.recordTransaction(order)
	}

	s.db.Save(order)
}

// executeSellOrder 执行卖出订单
func (s *Service) executeSellOrder(order *models.Order) {
	// 根据平台执行不同的出售逻辑
	var err error
	
	switch order.Platform {
	case "buff":
		if s.config.BuffAPI.Enabled {
			err = s.executeBuffSell(order)
		}
	case "youpin":
		if s.config.YouPin.Enabled {
			err = s.executeYouPinSell(order)
		}
	case "steam":
		err = s.executeSteamSell(order)
	default:
		err = errors.New("unsupported platform")
	}

	if err != nil {
		order.Status = "failed"
		order.FailedReason = err.Error()
		// 解锁库存
		s.unlockInventory(order.UserID, order.ItemID, order.Quantity)
	} else {
		order.Status = "completed"
		now := time.Now()
		order.ExecutedAt = &now
		
		// 从库存移除
		s.removeFromInventory(order)
		
		// 记录交易
		s.recordTransaction(order)
	}

	s.db.Save(order)
}

// GetStrategies 获取交易策略
func (s *Service) GetStrategies(userID uint) ([]models.Strategy, error) {
	var strategies []models.Strategy
	err := s.db.Where("user_id = ?", userID).Find(&strategies).Error
	return strategies, err
}

// CreateStrategy 创建交易策略
func (s *Service) CreateStrategy(userID uint, strategy *models.Strategy) error {
	strategy.UserID = userID
	strategy.Status = "paused"
	return s.db.Create(strategy).Error
}

// UpdateStrategy 更新交易策略
func (s *Service) UpdateStrategy(strategyID uint, userID uint, updates map[string]interface{}) error {
	return s.db.Model(&models.Strategy{}).
		Where("id = ? AND user_id = ?", strategyID, userID).
		Updates(updates).Error
}

// DeleteStrategy 删除交易策略
func (s *Service) DeleteStrategy(strategyID uint, userID uint) error {
	return s.db.Where("id = ? AND user_id = ?", strategyID, userID).
		Delete(&models.Strategy{}).Error
}

// ActivateStrategy 激活策略
func (s *Service) ActivateStrategy(strategyID uint, userID uint) error {
	var strategy models.Strategy
	if err := s.db.Where("id = ? AND user_id = ?", strategyID, userID).First(&strategy).Error; err != nil {
		return err
	}

	strategy.Status = "active"
	if err := s.db.Save(&strategy).Error; err != nil {
		return err
	}

	// 启动策略执行器
	go s.runStrategy(&strategy)

	return nil
}

// DeactivateStrategy 停用策略
func (s *Service) DeactivateStrategy(strategyID uint, userID uint) error {
	return s.db.Model(&models.Strategy{}).
		Where("id = ? AND user_id = ?", strategyID, userID).
		Update("status", "paused").Error
}

// runStrategy 运行策略
func (s *Service) runStrategy(strategy *models.Strategy) {
	ticker := time.NewTicker(1 * time.Minute) // 每分钟检查一次
	defer ticker.Stop()

	for range ticker.C {
		// 检查策略是否仍然激活
		var currentStrategy models.Strategy
		if err := s.db.First(&currentStrategy, strategy.ID).Error; err != nil {
			return
		}

		if currentStrategy.Status != "active" {
			return
		}

		// 根据策略类型执行不同的逻辑
		switch strategy.Type {
		case "grid":
			s.executeGridStrategy(strategy)
		case "arbitrage":
			s.executeArbitrageStrategy(strategy)
		case "trend_following":
			s.executeTrendFollowingStrategy(strategy)
		case "mean_reversion":
			s.executeMeanReversionStrategy(strategy)
		}
	}
}

// executeGridStrategy 执行网格策略
func (s *Service) executeGridStrategy(strategy *models.Strategy) {
	// 网格交易策略实现
	var config map[string]interface{}
	json.Unmarshal([]byte(strategy.Config), &config)
	
	// 获取价格区间和网格数量
	minPrice := config["min_price"].(float64)
	maxPrice := config["max_price"].(float64)
	gridCount := int(config["grid_count"].(float64))
	
	// 计算每个网格的价格
	gridSize := (maxPrice - minPrice) / float64(gridCount)
	
	// 检查当前价格并执行相应操作
	// 这里需要实现具体的网格交易逻辑
}

// executeArbitrageStrategy 执行套利策略
func (s *Service) executeArbitrageStrategy(strategy *models.Strategy) {
	// 套利策略实现
	// 比较不同平台的价格差异，寻找套利机会
}

// executeTrendFollowingStrategy 执行趋势跟踪策略
func (s *Service) executeTrendFollowingStrategy(strategy *models.Strategy) {
	// 趋势跟踪策略实现
	// 根据移动平均线等指标判断趋势
}

// executeMeanReversionStrategy 执行均值回归策略
func (s *Service) executeMeanReversionStrategy(strategy *models.Strategy) {
	// 均值回归策略实现
	// 当价格偏离均值时进行交易
}

// GetProfitStats 获取盈利统计
func (s *Service) GetProfitStats(userID uint, period string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	var startDate time.Time
	switch period {
	case "day":
		startDate = time.Now().AddDate(0, 0, -1)
	case "week":
		startDate = time.Now().AddDate(0, 0, -7)
	case "month":
		startDate = time.Now().AddDate(0, -1, 0)
	case "year":
		startDate = time.Now().AddDate(-1, 0, 0)
	default:
		startDate = time.Now().AddDate(0, -1, 0)
	}

	// 计算总盈利
	var totalProfit float64
	s.db.Model(&models.Transaction{}).
		Where("user_id = ? AND completed_at >= ?", userID, startDate).
		Select("SUM(profit)").Scan(&totalProfit)
	stats["total_profit"] = totalProfit

	// 计算交易次数
	var tradeCount int64
	s.db.Model(&models.Transaction{}).
		Where("user_id = ? AND completed_at >= ?", userID, startDate).
		Count(&tradeCount)
	stats["trade_count"] = tradeCount

	// 计算胜率
	var winCount int64
	s.db.Model(&models.Transaction{}).
		Where("user_id = ? AND completed_at >= ? AND profit > 0", userID, startDate).
		Count(&winCount)
	
	winRate := 0.0
	if tradeCount > 0 {
		winRate = float64(winCount) / float64(tradeCount) * 100
	}
	stats["win_rate"] = winRate

	// 最佳交易
	var bestTrade models.Transaction
	s.db.Where("user_id = ? AND completed_at >= ?", userID, startDate).
		Order("profit DESC").First(&bestTrade)
	stats["best_trade"] = bestTrade

	return stats, nil
}

// GetTradingStats 获取交易统计
func (s *Service) GetTradingStats(userID uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// 总交易量
	var totalVolume float64
	s.db.Model(&models.Transaction{}).
		Where("user_id = ?", userID).
		Select("SUM(amount)").Scan(&totalVolume)
	stats["total_volume"] = totalVolume

	// 活跃订单数
	var activeOrders int64
	s.db.Model(&models.Order{}).
		Where("user_id = ? AND status = ?", userID, "pending").
		Count(&activeOrders)
	stats["active_orders"] = activeOrders

	// 库存价值
	var inventoryValue float64
	s.db.Raw(`
		SELECT SUM(i.quantity * items.current_price) 
		FROM inventories i
		JOIN items ON i.item_id = items.id
		WHERE i.user_id = ?
	`, userID).Scan(&inventoryValue)
	stats["inventory_value"] = inventoryValue

	// 策略数量
	var strategyCount int64
	s.db.Model(&models.Strategy{}).
		Where("user_id = ?", userID).
		Count(&strategyCount)
	stats["strategy_count"] = strategyCount

	return stats, nil
}

// 辅助函数
func (s *Service) checkUserBalance(userID uint, amount float64) bool {
	// 实际实现需要接入支付系统
	return true
}

func (s *Service) checkInventory(userID uint, itemID uint, quantity int) bool {
	var count int64
	s.db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ? AND quantity >= ? AND locked = ?", 
			userID, itemID, quantity, false).
		Count(&count)
	return count > 0
}

func (s *Service) lockInventory(userID uint, itemID uint, quantity int) error {
	return s.db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ?", userID, itemID).
		Update("locked", true).Error
}

func (s *Service) unlockInventory(userID uint, itemID uint, quantity int) error {
	return s.db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ?", userID, itemID).
		Update("locked", false).Error
}

func (s *Service) addToInventory(order *models.Order) {
	inventory := models.Inventory{
		UserID:     order.UserID,
		ItemID:     order.ItemID,
		Quantity:   order.Quantity,
		BuyPrice:   order.Price,
		Platform:   order.Platform,
		AcquiredAt: time.Now(),
		Tradable:   true,
	}
	s.db.Create(&inventory)
}

func (s *Service) removeFromInventory(order *models.Order) {
	s.db.Where("user_id = ? AND item_id = ?", order.UserID, order.ItemID).
		Delete(&models.Inventory{})
}

func (s *Service) recordTransaction(order *models.Order) {
	transaction := models.Transaction{
		UserID:      order.UserID,
		OrderID:     order.ID,
		Type:        order.Type,
		Amount:      order.Price * float64(order.Quantity),
		Platform:    order.Platform,
		CompletedAt: time.Now(),
	}
	
	// 计算手续费（简化处理）
	transaction.Fee = transaction.Amount * 0.025 // 2.5%手续费
	
	// 如果是卖单，计算利润
	if order.Type == "sell" {
		var buyPrice float64
		s.db.Model(&models.Inventory{}).
			Where("user_id = ? AND item_id = ?", order.UserID, order.ItemID).
			Select("buy_price").Scan(&buyPrice)
		transaction.Profit = (order.Price - buyPrice) * float64(order.Quantity) - transaction.Fee
	}
	
	s.db.Create(&transaction)
}

// Platform specific implementations (需要根据实际API实现)
func (s *Service) executeBuffBuy(order *models.Order) error {
	// BUFF平台购买实现
	return nil
}

func (s *Service) executeBuffSell(order *models.Order) error {
	// BUFF平台出售实现
	return nil
}

func (s *Service) executeYouPinBuy(order *models.Order) error {
	// 悠悠有品购买实现
	return nil
}

func (s *Service) executeYouPinSell(order *models.Order) error {
	// 悠悠有品出售实现
	return nil
}

func (s *Service) executeSteamBuy(order *models.Order) error {
	// Steam市场购买实现
	return nil
}

func (s *Service) executeSteamSell(order *models.Order) error {
	// Steam市场出售实现
	return nil
}