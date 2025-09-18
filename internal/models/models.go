package models

import (
	"time"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	SteamID     string         `json:"steam_id" gorm:"unique;not null"`
	Username    string         `json:"username"`
	Avatar      string         `json:"avatar"`
	AccessToken string         `json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// Item represents a CSGO item
type Item struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null"`
	MarketName  string         `json:"market_name" gorm:"unique;not null"`
	IconURL     string         `json:"icon_url"`
	Type        string         `json:"type"`
	Weapon      string         `json:"weapon"`
	Exterior    string         `json:"exterior"`
	Rarity      string         `json:"rarity"`
	Collection  string         `json:"collection"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// Price represents price data for an item on different platforms
type Price struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	ItemID     uint      `json:"item_id" gorm:"not null"`
	Item       Item      `json:"item" gorm:"foreignKey:ItemID"`
	Platform   string    `json:"platform" gorm:"not null"` // steam, buff, youpin
	Price      float64   `json:"price"`
	Volume     int       `json:"volume"`
	Currency   string    `json:"currency" gorm:"default:'USD'"`
	Timestamp  time.Time `json:"timestamp"`
	CreatedAt  time.Time `json:"created_at"`
}

// Trade represents a trading transaction
type Trade struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UserID      uint           `json:"user_id" gorm:"not null"`
	User        User           `json:"user" gorm:"foreignKey:UserID"`
	ItemID      uint           `json:"item_id" gorm:"not null"`
	Item        Item           `json:"item" gorm:"foreignKey:ItemID"`
	Platform    string         `json:"platform" gorm:"not null"`
	Type        string         `json:"type" gorm:"not null"` // buy, sell
	Price       float64        `json:"price"`
	Quantity    int            `json:"quantity" gorm:"default:1"`
	Status      string         `json:"status" gorm:"default:'pending'"` // pending, completed, failed, cancelled
	TradeID     string         `json:"trade_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// Strategy represents a trading strategy
type Strategy struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UserID      uint           `json:"user_id" gorm:"not null"`
	User        User           `json:"user" gorm:"foreignKey:UserID"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`
	ItemID      uint           `json:"item_id"`
	Item        Item           `json:"item" gorm:"foreignKey:ItemID"`
	BuyPrice    float64        `json:"buy_price"`
	SellPrice   float64        `json:"sell_price"`
	MaxQuantity int            `json:"max_quantity" gorm:"default:1"`
	IsActive    bool           `json:"is_active" gorm:"default:false"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// Inventory represents user's inventory items
type Inventory struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	ItemID    uint      `json:"item_id" gorm:"not null"`
	Item      Item      `json:"item" gorm:"foreignKey:ItemID"`
	Platform  string    `json:"platform" gorm:"not null"`
	AssetID   string    `json:"asset_id"`
	Quantity  int       `json:"quantity" gorm:"default:1"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MarketTrend represents market trend analysis
type MarketTrend struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	ItemID          uint      `json:"item_id" gorm:"not null"`
	Item            Item      `json:"item" gorm:"foreignKey:ItemID"`
	Platform        string    `json:"platform" gorm:"not null"`
	TrendDirection  string    `json:"trend_direction"` // up, down, stable
	PriceChange     float64   `json:"price_change"`
	VolumeChange    float64   `json:"volume_change"`
	Confidence      float64   `json:"confidence"`
	AnalysisDate    time.Time `json:"analysis_date"`
	CreatedAt       time.Time `json:"created_at"`
}