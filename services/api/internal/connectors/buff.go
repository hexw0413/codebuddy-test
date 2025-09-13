package connectors

import "context"

// BuffClient is a stub client for BUFF market integration.
type BuffClient struct {
    apiKey string
}

func NewBuffClient(apiKey string) *BuffClient {
    return &BuffClient{apiKey: apiKey}
}

func (c *BuffClient) GetInventory(ctx context.Context, steamID string) ([]Item, error) {
    return []Item{}, nil
}

func (c *BuffClient) BuyItem(ctx context.Context, itemID string, price float64) (string, error) {
    return "order-id-stub", nil
}

func (c *BuffClient) SellItem(ctx context.Context, itemID string, price float64) (string, error) {
    return "order-id-stub", nil
}

type Item struct {
    ID    string
    Name  string
    Price float64
}

