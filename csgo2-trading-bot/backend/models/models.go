package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	gorm.Model
	SteamID          string    `json:"steam_id" gorm:"unique;not null"`
	Username         string    `json:"username"`
	Avatar           string    `json:"avatar"`
	TradeURL         string    `json:"trade_url"`
	APIKey           string    `json:"-"`
	SharedSecret     string    `json:"-"`
	IdentitySecret   string    `json:"-"`
	LastLogin        time.Time `json:"last_login"`
	TotalProfit      float64   `json:"total_profit"`
	TotalTransactions int      `json:"total_transactions"`
}

// Item 物品模型
type Item struct {
	gorm.Model
	MarketHashName string  `json:"market_hash_name" gorm:"unique;not null"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Rarity         string  `json:"rarity"`
	Quality        string  `json:"quality"`
	IconURL        string  `json:"icon_url"`
	CurrentPrice   float64 `json:"current_price"`
	AvgPrice7Days  float64 `json:"avg_price_7days"`
	AvgPrice30Days float64 `json:"avg_price_30days"`
	Volume24h      int     `json:"volume_24h"`
	LastUpdated    time.Time `json:"last_updated"`
}

// PriceHistory 价格历史
type PriceHistory struct {
	gorm.Model
	ItemID       uint      `json:"item_id"`
	Item         Item      `json:"item" gorm:"foreignKey:ItemID"`
	Price        float64   `json:"price"`
	Volume       int       `json:"volume"`
	Platform     string    `json:"platform"` // buff, youpin, steam
	RecordedAt   time.Time `json:"recorded_at"`
}

// Order 订单模型
type Order struct {
	gorm.Model
	UserID       uint      `json:"user_id"`
	User         User      `json:"user" gorm:"foreignKey:UserID"`
	ItemID       uint      `json:"item_id"`
	Item         Item      `json:"item" gorm:"foreignKey:ItemID"`
	Type         string    `json:"type"` // buy, sell
	Status       string    `json:"status"` // pending, completed, cancelled, failed
	Price        float64   `json:"price"`
	Quantity     int       `json:"quantity"`
	Platform     string    `json:"platform"`
	StrategyID   *uint     `json:"strategy_id,omitempty"`
	Strategy     *Strategy `json:"strategy,omitempty" gorm:"foreignKey:StrategyID"`
	ExecutedAt   *time.Time `json:"executed_at,omitempty"`
	FailedReason string    `json:"failed_reason,omitempty"`
}

// Transaction 交易记录
type Transaction struct {
	gorm.Model
	UserID      uint    `json:"user_id"`
	User        User    `json:"user" gorm:"foreignKey:UserID"`
	OrderID     uint    `json:"order_id"`
	Order       Order   `json:"order" gorm:"foreignKey:OrderID"`
	Type        string  `json:"type"` // buy, sell
	Amount      float64 `json:"amount"`
	Fee         float64 `json:"fee"`
	Profit      float64 `json:"profit"`
	Platform    string  `json:"platform"`
	TradeID     string  `json:"trade_id"`
	CompletedAt time.Time `json:"completed_at"`
}

// Strategy 交易策略
type Strategy struct {
	gorm.Model
	UserID      uint    `json:"user_id"`
	User        User    `json:"user" gorm:"foreignKey:UserID"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        string  `json:"type"` // grid, arbitrage, trend_following, mean_reversion
	Status      string  `json:"status"` // active, paused, stopped
	Config      string  `json:"config" gorm:"type:jsonb"` // JSON配置
	MaxInvest   float64 `json:"max_invest"`
	MinProfit   float64 `json:"min_profit"`
	StopLoss    float64 `json:"stop_loss"`
	TakeProfit  float64 `json:"take_profit"`
	Performance string  `json:"performance" gorm:"type:jsonb"` // 性能统计JSON
}

// Inventory 库存
type Inventory struct {
	gorm.Model
	UserID     uint      `json:"user_id"`
	User       User      `json:"user" gorm:"foreignKey:UserID"`
	ItemID     uint      `json:"item_id"`
	Item       Item      `json:"item" gorm:"foreignKey:ItemID"`
	AssetID    string    `json:"asset_id"`
	Quantity   int       `json:"quantity"`
	BuyPrice   float64   `json:"buy_price"`
	Platform   string    `json:"platform"`
	AcquiredAt time.Time `json:"acquired_at"`
	Tradable   bool      `json:"tradable"`
	Locked     bool      `json:"locked"` // 是否被策略锁定
}

// MarketData 市场数据快照
type MarketData struct {
	gorm.Model
	ItemID         uint    `json:"item_id"`
	Item           Item    `json:"item" gorm:"foreignKey:ItemID"`
	Platform       string  `json:"platform"`
	LowestPrice    float64 `json:"lowest_price"`
	HighestPrice   float64 `json:"highest_price"`
	MedianPrice    float64 `json:"median_price"`
	Volume         int     `json:"volume"`
	BuyOrders      int     `json:"buy_orders"`
	SellOrders     int     `json:"sell_orders"`
	PriceChange24h float64 `json:"price_change_24h"`
	SnapshotTime   time.Time `json:"snapshot_time"`
}

// Notification 通知
type Notification struct {
	gorm.Model
	UserID   uint      `json:"user_id"`
	User     User      `json:"user" gorm:"foreignKey:UserID"`
	Type     string    `json:"type"` // price_alert, order_executed, strategy_alert
	Title    string    `json:"title"`
	Message  string    `json:"message"`
	Read     bool      `json:"read"`
	Priority string    `json:"priority"` // low, medium, high
	Data     string    `json:"data" gorm:"type:jsonb"`
	ReadAt   *time.Time `json:"read_at,omitempty"`
}