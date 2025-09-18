package services

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"csgo-trader/internal/models"
)

type BuffService struct {
	apiKey string
	client *resty.Client
	baseURL string
}

type BuffMarketItem struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	ShortName   string  `json:"short_name"`
	SellMinPrice string `json:"sell_min_price"`
	BuyMaxPrice  string `json:"buy_max_price"`
	SellNum      int    `json:"sell_num"`
	BuyNum       int    `json:"buy_num"`
	IconURL      string `json:"icon_url"`
	GoodsInfo    struct {
		Info struct {
			Tags struct {
				Type struct {
					Category string `json:"category"`
					Name     string `json:"name"`
				} `json:"type"`
				Weapon struct {
					Category string `json:"category"`
					Name     string `json:"name"`
				} `json:"weapon"`
				Exterior struct {
					Category string `json:"category"`
					Name     string `json:"name"`
				} `json:"exterior"`
				Rarity struct {
					Category string `json:"category"`
					Name     string `json:"name"`
				} `json:"rarity"`
			} `json:"tags"`
		} `json:"info"`
	} `json:"goods_info"`
}

type BuffResponse struct {
	Code string `json:"code"`
	Data struct {
		Items      []BuffMarketItem `json:"items"`
		TotalCount int              `json:"total_count"`
		TotalPage  int              `json:"total_page"`
	} `json:"data"`
	Msg string `json:"msg"`
}

type BuffInventoryItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Price       string  `json:"price"`
	AssetInfo   struct {
		AssetID     string `json:"assetid"`
		ContextID   string `json:"contextid"`
		MarketName  string `json:"market_name"`
		IconURL     string `json:"icon_url"`
		Tradable    bool   `json:"tradable"`
		Marketable  bool   `json:"marketable"`
	} `json:"asset_info"`
}

type BuffInventoryResponse struct {
	Code string `json:"code"`
	Data struct {
		Items []BuffInventoryItem `json:"items"`
	} `json:"data"`
	Msg string `json:"msg"`
}

func NewBuffService(apiKey string) *BuffService {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("User-Agent", "CSGO-Trader/1.0")
	
	return &BuffService{
		apiKey:  apiKey,
		client:  client,
		baseURL: "https://buff.163.com/api",
	}
}

func (b *BuffService) GetMarketItems(page int, limit int) ([]BuffMarketItem, error) {
	url := fmt.Sprintf("%s/market/goods", b.baseURL)
	
	resp, err := b.client.R().
		SetQueryParams(map[string]string{
			"game":     "csgo",
			"page_num": strconv.Itoa(page),
			"page_size": strconv.Itoa(limit),
		}).
		SetHeader("Cookie", fmt.Sprintf("session=%s", b.apiKey)).
		Get(url)
	
	if err != nil {
		return nil, err
	}

	var buffResp BuffResponse
	if err := json.Unmarshal(resp.Body(), &buffResp); err != nil {
		return nil, err
	}

	if buffResp.Code != "OK" {
		return nil, fmt.Errorf("buff API error: %s", buffResp.Msg)
	}

	return buffResp.Data.Items, nil
}

func (b *BuffService) GetItemPrice(itemName string) (*models.Price, error) {
	url := fmt.Sprintf("%s/market/goods", b.baseURL)
	
	resp, err := b.client.R().
		SetQueryParams(map[string]string{
			"game":   "csgo",
			"search": itemName,
		}).
		SetHeader("Cookie", fmt.Sprintf("session=%s", b.apiKey)).
		Get(url)
	
	if err != nil {
		return nil, err
	}

	var buffResp BuffResponse
	if err := json.Unmarshal(resp.Body(), &buffResp); err != nil {
		return nil, err
	}

	if buffResp.Code != "OK" {
		return nil, fmt.Errorf("buff API error: %s", buffResp.Msg)
	}

	if len(buffResp.Data.Items) == 0 {
		return nil, fmt.Errorf("item not found")
	}

	item := buffResp.Data.Items[0]
	
	// Parse price from string
	priceStr := item.SellMinPrice
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return nil, err
	}

	return &models.Price{
		Platform:  "buff",
		Price:     price,
		Volume:    item.SellNum,
		Currency:  "CNY",
		Timestamp: time.Now(),
	}, nil
}

func (b *BuffService) GetUserInventory(userID string) ([]BuffInventoryItem, error) {
	url := fmt.Sprintf("%s/market/steam_inventory", b.baseURL)
	
	resp, err := b.client.R().
		SetQueryParams(map[string]string{
			"game":    "csgo",
			"user_id": userID,
		}).
		SetHeader("Cookie", fmt.Sprintf("session=%s", b.apiKey)).
		Get(url)
	
	if err != nil {
		return nil, err
	}

	var buffResp BuffInventoryResponse
	if err := json.Unmarshal(resp.Body(), &buffResp); err != nil {
		return nil, err
	}

	if buffResp.Code != "OK" {
		return nil, fmt.Errorf("buff API error: %s", buffResp.Msg)
	}

	return buffResp.Data.Items, nil
}

func (b *BuffService) BuyItem(itemID string, price float64) error {
	url := fmt.Sprintf("%s/market/goods/buy", b.baseURL)
	
	resp, err := b.client.R().
		SetFormData(map[string]string{
			"goods_id": itemID,
			"price":    fmt.Sprintf("%.2f", price),
		}).
		SetHeader("Cookie", fmt.Sprintf("session=%s", b.apiKey)).
		Post(url)
	
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return err
	}

	if code, ok := result["code"].(string); !ok || code != "OK" {
		msg := "unknown error"
		if m, ok := result["msg"].(string); ok {
			msg = m
		}
		return fmt.Errorf("buy failed: %s", msg)
	}

	return nil
}

func (b *BuffService) SellItem(assetID string, price float64) error {
	url := fmt.Sprintf("%s/market/goods/sell", b.baseURL)
	
	resp, err := b.client.R().
		SetFormData(map[string]string{
			"assetid": assetID,
			"price":   fmt.Sprintf("%.2f", price),
			"game":    "csgo",
		}).
		SetHeader("Cookie", fmt.Sprintf("session=%s", b.apiKey)).
		Post(url)
	
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return err
	}

	if code, ok := result["code"].(string); !ok || code != "OK" {
		msg := "unknown error"
		if m, ok := result["msg"].(string); ok {
			msg = m
		}
		return fmt.Errorf("sell failed: %s", msg)
	}

	return nil
}

func (b *BuffService) GetPriceHistory(itemID string, days int) ([]models.Price, error) {
	url := fmt.Sprintf("%s/market/goods/price_history", b.baseURL)
	
	resp, err := b.client.R().
		SetQueryParams(map[string]string{
			"goods_id": itemID,
			"days":     strconv.Itoa(days),
		}).
		SetHeader("Cookie", fmt.Sprintf("session=%s", b.apiKey)).
		Get(url)
	
	if err != nil {
		return nil, err
	}

	var result struct {
		Code string `json:"code"`
		Data struct {
			PriceHistory [][]interface{} `json:"price_history"`
		} `json:"data"`
		Msg string `json:"msg"`
	}

	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}

	if result.Code != "OK" {
		return nil, fmt.Errorf("buff API error: %s", result.Msg)
	}

	var prices []models.Price
	for _, entry := range result.Data.PriceHistory {
		if len(entry) >= 2 {
			timestamp := time.Unix(int64(entry[0].(float64)), 0)
			price := entry[1].(float64)
			
			prices = append(prices, models.Price{
				Platform:  "buff",
				Price:     price,
				Currency:  "CNY",
				Timestamp: timestamp,
			})
		}
	}

	return prices, nil
}