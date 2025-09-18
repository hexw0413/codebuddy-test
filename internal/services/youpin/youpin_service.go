package services

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"csgo-trader/internal/models"
)

type YoupinService struct {
	apiKey string
	client *resty.Client
	baseURL string
}

type YoupinMarketItem struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Volume      int     `json:"volume"`
	IconURL     string  `json:"icon_url"`
	Type        string  `json:"type"`
	Weapon      string  `json:"weapon"`
	Exterior    string  `json:"exterior"`
	Rarity      string  `json:"rarity"`
	Collection  string  `json:"collection"`
}

type YoupinResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Items      []YoupinMarketItem `json:"items"`
		TotalCount int                `json:"total_count"`
		TotalPage  int                `json:"total_page"`
	} `json:"data"`
}

type YoupinInventoryItem struct {
	ID         string  `json:"id"`
	AssetID    string  `json:"asset_id"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	IconURL    string  `json:"icon_url"`
	Tradable   bool    `json:"tradable"`
	Marketable bool    `json:"marketable"`
}

type YoupinInventoryResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Items []YoupinInventoryItem `json:"items"`
	} `json:"data"`
}

func NewYoupinService(apiKey string) *YoupinService {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("User-Agent", "CSGO-Trader/1.0")
	client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	
	return &YoupinService{
		apiKey:  apiKey,
		client:  client,
		baseURL: "https://api.youpin898.com/api/v1",
	}
}

func (y *YoupinService) GetMarketItems(page int, limit int) ([]YoupinMarketItem, error) {
	url := fmt.Sprintf("%s/market/items", y.baseURL)
	
	resp, err := y.client.R().
		SetQueryParams(map[string]string{
			"game":     "csgo",
			"page":     strconv.Itoa(page),
			"per_page": strconv.Itoa(limit),
		}).
		Get(url)
	
	if err != nil {
		return nil, err
	}

	var youpinResp YoupinResponse
	if err := json.Unmarshal(resp.Body(), &youpinResp); err != nil {
		return nil, err
	}

	if youpinResp.Code != 0 {
		return nil, fmt.Errorf("youpin API error: %s", youpinResp.Msg)
	}

	return youpinResp.Data.Items, nil
}

func (y *YoupinService) GetItemPrice(itemName string) (*models.Price, error) {
	url := fmt.Sprintf("%s/market/search", y.baseURL)
	
	resp, err := y.client.R().
		SetQueryParams(map[string]string{
			"game":    "csgo",
			"keyword": itemName,
		}).
		Get(url)
	
	if err != nil {
		return nil, err
	}

	var youpinResp YoupinResponse
	if err := json.Unmarshal(resp.Body(), &youpinResp); err != nil {
		return nil, err
	}

	if youpinResp.Code != 0 {
		return nil, fmt.Errorf("youpin API error: %s", youpinResp.Msg)
	}

	if len(youpinResp.Data.Items) == 0 {
		return nil, fmt.Errorf("item not found")
	}

	item := youpinResp.Data.Items[0]

	return &models.Price{
		Platform:  "youpin",
		Price:     item.Price,
		Volume:    item.Volume,
		Currency:  "CNY",
		Timestamp: time.Now(),
	}, nil
}

func (y *YoupinService) GetUserInventory(userID string) ([]YoupinInventoryItem, error) {
	url := fmt.Sprintf("%s/user/inventory", y.baseURL)
	
	resp, err := y.client.R().
		SetQueryParams(map[string]string{
			"game":    "csgo",
			"user_id": userID,
		}).
		Get(url)
	
	if err != nil {
		return nil, err
	}

	var youpinResp YoupinInventoryResponse
	if err := json.Unmarshal(resp.Body(), &youpinResp); err != nil {
		return nil, err
	}

	if youpinResp.Code != 0 {
		return nil, fmt.Errorf("youpin API error: %s", youpinResp.Msg)
	}

	return youpinResp.Data.Items, nil
}

func (y *YoupinService) BuyItem(itemID string, price float64) error {
	url := fmt.Sprintf("%s/market/buy", y.baseURL)
	
	resp, err := y.client.R().
		SetFormData(map[string]string{
			"item_id": itemID,
			"price":   fmt.Sprintf("%.2f", price),
		}).
		Post(url)
	
	if err != nil {
		return err
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return err
	}

	if result.Code != 0 {
		return fmt.Errorf("buy failed: %s", result.Msg)
	}

	return nil
}

func (y *YoupinService) SellItem(assetID string, price float64) error {
	url := fmt.Sprintf("%s/market/sell", y.baseURL)
	
	resp, err := y.client.R().
		SetFormData(map[string]string{
			"asset_id": assetID,
			"price":    fmt.Sprintf("%.2f", price),
		}).
		Post(url)
	
	if err != nil {
		return err
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return err
	}

	if result.Code != 0 {
		return fmt.Errorf("sell failed: %s", result.Msg)
	}

	return nil
}

func (y *YoupinService) GetPriceHistory(itemID string, days int) ([]models.Price, error) {
	url := fmt.Sprintf("%s/market/history", y.baseURL)
	
	resp, err := y.client.R().
		SetQueryParams(map[string]string{
			"item_id": itemID,
			"days":    strconv.Itoa(days),
		}).
		Get(url)
	
	if err != nil {
		return nil, err
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			History []struct {
				Price     float64 `json:"price"`
				Volume    int     `json:"volume"`
				Timestamp int64   `json:"timestamp"`
			} `json:"history"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("youpin API error: %s", result.Msg)
	}

	var prices []models.Price
	for _, entry := range result.Data.History {
		timestamp := time.Unix(entry.Timestamp, 0)
		
		prices = append(prices, models.Price{
			Platform:  "youpin",
			Price:     entry.Price,
			Volume:    entry.Volume,
			Currency:  "CNY",
			Timestamp: timestamp,
		})
	}

	return prices, nil
}