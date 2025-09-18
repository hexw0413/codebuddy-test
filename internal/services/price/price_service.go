package services

import (
	"log"
	"time"

	"gorm.io/gorm"
	"csgo-trader/internal/models"
	steamService "csgo-trader/internal/services/steam"
	buffService "csgo-trader/internal/services/buff"
	youpinService "csgo-trader/internal/services/youpin"
)

type PriceService struct {
	db            *gorm.DB
	steamService  *steamService.SteamService
	buffService   *buffService.BuffService
	youpinService *youpinService.YoupinService
}

type PricePoint struct {
	Time     time.Time `json:"time"`
	Price    float64   `json:"price"`
	Volume   int       `json:"volume"`
	Platform string    `json:"platform"`
}

type PriceChart struct {
	ItemName string       `json:"item_name"`
	Data     []PricePoint `json:"data"`
}

func NewPriceService(db *gorm.DB) *PriceService {
	return &PriceService{
		db: db,
	}
}

func (p *PriceService) SetServices(steam *steamService.SteamService, buff *buffService.BuffService, youpin *youpinService.YoupinService) {
	p.steamService = steam
	p.buffService = buff
	p.youpinService = youpin
}

func (p *PriceService) SavePrice(price *models.Price) error {
	return p.db.Create(price).Error
}

func (p *PriceService) GetPriceHistory(itemID uint, platform string, days int) ([]models.Price, error) {
	var prices []models.Price
	
	query := p.db.Where("item_id = ?", itemID)
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	
	since := time.Now().AddDate(0, 0, -days)
	query = query.Where("timestamp >= ?", since)
	
	err := query.Order("timestamp ASC").Find(&prices).Error
	return prices, err
}

func (p *PriceService) GetLatestPrices(itemID uint) (map[string]*models.Price, error) {
	var prices []models.Price
	
	// Get latest price for each platform
	err := p.db.Raw(`
		SELECT * FROM prices 
		WHERE item_id = ? AND timestamp IN (
			SELECT MAX(timestamp) 
			FROM prices 
			WHERE item_id = ? 
			GROUP BY platform
		)
	`, itemID, itemID).Find(&prices).Error
	
	if err != nil {
		return nil, err
	}
	
	result := make(map[string]*models.Price)
	for i, price := range prices {
		result[price.Platform] = &prices[i]
	}
	
	return result, nil
}

func (p *PriceService) GetPriceChart(itemID uint, days int) (*PriceChart, error) {
	var item models.Item
	if err := p.db.First(&item, itemID).Error; err != nil {
		return nil, err
	}
	
	prices, err := p.GetPriceHistory(itemID, "", days)
	if err != nil {
		return nil, err
	}
	
	var dataPoints []PricePoint
	for _, price := range prices {
		dataPoints = append(dataPoints, PricePoint{
			Time:     price.Timestamp,
			Price:    price.Price,
			Volume:   price.Volume,
			Platform: price.Platform,
		})
	}
	
	return &PriceChart{
		ItemName: item.MarketName,
		Data:     dataPoints,
	}, nil
}

func (p *PriceService) CalculateTrend(itemID uint, platform string, days int) (*models.MarketTrend, error) {
	prices, err := p.GetPriceHistory(itemID, platform, days)
	if err != nil {
		return nil, err
	}
	
	if len(prices) < 2 {
		return nil, nil
	}
	
	// Simple trend calculation
	firstPrice := prices[0].Price
	lastPrice := prices[len(prices)-1].Price
	priceChange := lastPrice - firstPrice
	priceChangePercent := (priceChange / firstPrice) * 100
	
	// Calculate volume change
	firstVolume := float64(prices[0].Volume)
	lastVolume := float64(prices[len(prices)-1].Volume)
	volumeChange := ((lastVolume - firstVolume) / firstVolume) * 100
	
	// Determine trend direction
	var trendDirection string
	var confidence float64
	
	if priceChangePercent > 5 {
		trendDirection = "up"
		confidence = 0.8
	} else if priceChangePercent < -5 {
		trendDirection = "down"
		confidence = 0.8
	} else {
		trendDirection = "stable"
		confidence = 0.6
	}
	
	trend := &models.MarketTrend{
		ItemID:         itemID,
		Platform:       platform,
		TrendDirection: trendDirection,
		PriceChange:    priceChangePercent,
		VolumeChange:   volumeChange,
		Confidence:     confidence,
		AnalysisDate:   time.Now(),
	}
	
	// Save trend to database
	p.db.Create(trend)
	
	return trend, nil
}

func (p *PriceService) CollectPrices() {
	log.Println("Starting price collection...")
	
	for {
		// Get all items to collect prices for
		var items []models.Item
		if err := p.db.Find(&items).Error; err != nil {
			log.Printf("Failed to get items: %v", err)
			time.Sleep(5 * time.Minute)
			continue
		}
		
		for _, item := range items {
			p.collectItemPrices(&item)
			time.Sleep(2 * time.Second) // Rate limiting
		}
		
		log.Printf("Collected prices for %d items", len(items))
		time.Sleep(15 * time.Minute) // Collect every 15 minutes
	}
}

func (p *PriceService) collectItemPrices(item *models.Item) {
	// Collect from Steam
	if p.steamService != nil {
		if steamPrice, err := p.steamService.GetMarketPrice(item.MarketName); err == nil {
			steamPrice.ItemID = item.ID
			p.SavePrice(steamPrice)
		}
	}
	
	// Collect from BUFF
	if p.buffService != nil {
		if buffPrice, err := p.buffService.GetItemPrice(item.MarketName); err == nil {
			buffPrice.ItemID = item.ID
			p.SavePrice(buffPrice)
		}
	}
	
	// Collect from YouPin
	if p.youpinService != nil {
		if youpinPrice, err := p.youpinService.GetItemPrice(item.MarketName); err == nil {
			youpinPrice.ItemID = item.ID
			p.SavePrice(youpinPrice)
		}
	}
}

func (p *PriceService) GetTopMovers(limit int) ([]models.MarketTrend, error) {
	var trends []models.MarketTrend
	
	err := p.db.Preload("Item").
		Where("analysis_date >= ?", time.Now().AddDate(0, 0, -1)).
		Order("ABS(price_change) DESC").
		Limit(limit).
		Find(&trends).Error
	
	return trends, err
}

func (p *PriceService) GetArbitrageOpportunities(minProfitPercent float64) ([]ArbitrageOpportunity, error) {
	var opportunities []ArbitrageOpportunity
	
	// Get all items with recent prices on multiple platforms
	var items []models.Item
	if err := p.db.Find(&items).Error; err != nil {
		return nil, err
	}
	
	for _, item := range items {
		prices, err := p.GetLatestPrices(item.ID)
		if err != nil {
			continue
		}
		
		if len(prices) < 2 {
			continue
		}
		
		// Find arbitrage opportunities
		for platform1, price1 := range prices {
			for platform2, price2 := range prices {
				if platform1 != platform2 && price1.Price < price2.Price {
					profit := price2.Price - price1.Price
					profitPercent := (profit / price1.Price) * 100
					
					if profitPercent >= minProfitPercent {
						opportunities = append(opportunities, ArbitrageOpportunity{
							Item:           item,
							BuyPlatform:    platform1,
							SellPlatform:   platform2,
							BuyPrice:       price1.Price,
							SellPrice:      price2.Price,
							Profit:         profit,
							ProfitPercent:  profitPercent,
						})
					}
				}
			}
		}
	}
	
	return opportunities, nil
}

type ArbitrageOpportunity struct {
	Item          models.Item `json:"item"`
	BuyPlatform   string      `json:"buy_platform"`
	SellPlatform  string      `json:"sell_platform"`
	BuyPrice      float64     `json:"buy_price"`
	SellPrice     float64     `json:"sell_price"`
	Profit        float64     `json:"profit"`
	ProfitPercent float64     `json:"profit_percent"`
}