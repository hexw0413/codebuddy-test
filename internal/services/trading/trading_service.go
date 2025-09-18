package services

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"csgo-trader/internal/models"
	steamService "csgo-trader/internal/services/steam"
	buffService "csgo-trader/internal/services/buff"
	youpinService "csgo-trader/internal/services/youpin"
)

type TradingService struct {
	db            *gorm.DB
	steamService  *steamService.SteamService
	buffService   *buffService.BuffService
	youpinService *youpinService.YoupinService
}

type TradeSignal struct {
	ItemID    uint
	Platform  string
	Action    string  // "buy" or "sell"
	Price     float64
	Confidence float64
	Reason    string
}

func NewTradingService(db *gorm.DB, steam *steamService.SteamService, buff *buffService.BuffService, youpin *youpinService.YoupinService) *TradingService {
	return &TradingService{
		db:            db,
		steamService:  steam,
		buffService:   buff,
		youpinService: youpin,
	}
}

func (t *TradingService) ExecuteStrategy(strategyID uint) error {
	var strategy models.Strategy
	if err := t.db.Preload("Item").First(&strategy, strategyID).Error; err != nil {
		return err
	}

	if !strategy.IsActive {
		return fmt.Errorf("strategy is not active")
	}

	// Get current prices from all platforms
	prices, err := t.GetItemPrices(strategy.Item.MarketName)
	if err != nil {
		return err
	}

	// Analyze prices and generate trade signals
	signals := t.analyzePrices(prices, &strategy)

	// Execute trades based on signals
	for _, signal := range signals {
		if err := t.executeTrade(signal, &strategy); err != nil {
			log.Printf("Failed to execute trade: %v", err)
			continue
		}
	}

	return nil
}

func (t *TradingService) GetItemPrices(marketName string) (map[string]*models.Price, error) {
	prices := make(map[string]*models.Price)

	// Get Steam price
	if steamPrice, err := t.steamService.GetMarketPrice(marketName); err == nil {
		prices["steam"] = steamPrice
	}

	// Get BUFF price
	if buffPrice, err := t.buffService.GetItemPrice(marketName); err == nil {
		prices["buff"] = buffPrice
	}

	// Get YouPin price
	if youpinPrice, err := t.youpinService.GetItemPrice(marketName); err == nil {
		prices["youpin"] = youpinPrice
	}

	return prices, nil
}

func (t *TradingService) analyzePrices(prices map[string]*models.Price, strategy *models.Strategy) []TradeSignal {
	var signals []TradeSignal

	// Simple arbitrage strategy
	minPrice := float64(999999)
	maxPrice := float64(0)
	minPlatform := ""
	maxPlatform := ""

	for platform, price := range prices {
		if price.Price < minPrice {
			minPrice = price.Price
			minPlatform = platform
		}
		if price.Price > maxPrice {
			maxPrice = price.Price
			maxPlatform = platform
		}
	}

	// If price difference is significant, generate signals
	priceDiff := maxPrice - minPrice
	if priceDiff > 5.0 && priceDiff/minPrice > 0.1 { // 10% difference threshold
		// Buy from cheaper platform
		if minPrice <= strategy.BuyPrice {
			signals = append(signals, TradeSignal{
				ItemID:     strategy.ItemID,
				Platform:   minPlatform,
				Action:     "buy",
				Price:      minPrice,
				Confidence: 0.8,
				Reason:     fmt.Sprintf("Arbitrage opportunity: buy at %.2f, sell at %.2f", minPrice, maxPrice),
			})
		}

		// Sell to more expensive platform
		if maxPrice >= strategy.SellPrice {
			signals = append(signals, TradeSignal{
				ItemID:     strategy.ItemID,
				Platform:   maxPlatform,
				Action:     "sell",
				Price:      maxPrice,
				Confidence: 0.8,
				Reason:     fmt.Sprintf("Arbitrage opportunity: profit of %.2f", priceDiff),
			})
		}
	}

	return signals
}

func (t *TradingService) executeTrade(signal TradeSignal, strategy *models.Strategy) error {
	// Create trade record
	trade := models.Trade{
		UserID:   strategy.UserID,
		ItemID:   signal.ItemID,
		Platform: signal.Platform,
		Type:     signal.Action,
		Price:    signal.Price,
		Quantity: 1,
		Status:   "pending",
	}

	if err := t.db.Create(&trade).Error; err != nil {
		return err
	}

	// Execute the actual trade
	var err error
	switch signal.Platform {
	case "buff":
		if signal.Action == "buy" {
			err = t.buffService.BuyItem(fmt.Sprintf("%d", signal.ItemID), signal.Price)
		} else {
			// For selling, we need the asset ID from inventory
			err = fmt.Errorf("sell functionality requires asset ID implementation")
		}
	case "youpin":
		if signal.Action == "buy" {
			err = t.youpinService.BuyItem(fmt.Sprintf("%d", signal.ItemID), signal.Price)
		} else {
			err = fmt.Errorf("sell functionality requires asset ID implementation")
		}
	default:
		err = fmt.Errorf("platform %s not supported for automated trading", signal.Platform)
	}

	// Update trade status
	if err != nil {
		trade.Status = "failed"
		log.Printf("Trade failed: %v", err)
	} else {
		trade.Status = "completed"
		trade.TradeID = fmt.Sprintf("trade_%d_%d", time.Now().Unix(), trade.ID)
	}

	t.db.Save(&trade)
	return err
}

func (t *TradingService) GetUserTrades(userID uint, limit int) ([]models.Trade, error) {
	var trades []models.Trade
	err := t.db.Preload("Item").Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&trades).Error
	
	return trades, err
}

func (t *TradingService) GetActiveStrategies() ([]models.Strategy, error) {
	var strategies []models.Strategy
	err := t.db.Preload("Item").Preload("User").
		Where("is_active = ?", true).
		Find(&strategies).Error
	
	return strategies, err
}

func (t *TradingService) CreateStrategy(strategy *models.Strategy) error {
	return t.db.Create(strategy).Error
}

func (t *TradingService) UpdateStrategy(strategyID uint, updates map[string]interface{}) error {
	return t.db.Model(&models.Strategy{}).Where("id = ?", strategyID).Updates(updates).Error
}

func (t *TradingService) DeleteStrategy(strategyID uint) error {
	return t.db.Delete(&models.Strategy{}, strategyID).Error
}

func (t *TradingService) RunAutomatedTrading() {
	log.Println("Starting automated trading...")
	
	for {
		strategies, err := t.GetActiveStrategies()
		if err != nil {
			log.Printf("Failed to get active strategies: %v", err)
			time.Sleep(30 * time.Second)
			continue
		}

		for _, strategy := range strategies {
			if err := t.ExecuteStrategy(strategy.ID); err != nil {
				log.Printf("Failed to execute strategy %d: %v", strategy.ID, err)
			}
			time.Sleep(5 * time.Second) // Rate limiting
		}

		time.Sleep(60 * time.Second) // Run every minute
	}
}