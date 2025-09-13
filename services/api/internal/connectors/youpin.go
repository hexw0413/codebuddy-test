package connectors

import "context"

// YouPinClient is a stub client for 悠悠有品 integration.
type YouPinClient struct {
    apiKey string
}

func NewYouPinClient(apiKey string) *YouPinClient {
    return &YouPinClient{apiKey: apiKey}
}

func (c *YouPinClient) GetInventory(ctx context.Context, steamID string) ([]Item, error) {
    return []Item{}, nil
}

func (c *YouPinClient) BuyItem(ctx context.Context, itemID string, price float64) (string, error) {
    return "order-id-stub", nil
}

func (c *YouPinClient) SellItem(ctx context.Context, itemID string, price float64) (string, error) {
    return "order-id-stub", nil
}

