package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"csgo2-trading-bot/services/market"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应该限制
	},
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Println("Client connected")

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Println("Client disconnected")
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func HandleWebSocket(marketService *market.Service) gin.HandlerFunc {
	hub := NewHub()
	go hub.Run()

	// 启动价格更新推送
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// 获取最新市场数据
			trends, err := marketService.GetMarketTrends()
			if err != nil {
				continue
			}

			message := Message{
				Type: "market_update",
				Data: trends,
			}

			data, err := json.Marshal(message)
			if err != nil {
				continue
			}

			hub.broadcast <- data
		}
	}()

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println("WebSocket upgrade failed:", err)
			return
		}

		client := &Client{
			hub:  hub,
			conn: conn,
			send: make(chan []byte, 256),
		}

		client.hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// 处理客户端消息
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// 根据消息类型处理
		switch msg.Type {
		case "subscribe":
			// 处理订阅请求
			log.Printf("Client subscribed to: %v", msg.Data)
		case "unsubscribe":
			// 处理取消订阅请求
			log.Printf("Client unsubscribed from: %v", msg.Data)
		case "ping":
			// 响应ping
			response := Message{
				Type: "pong",
				Data: time.Now().Unix(),
			}
			data, _ := json.Marshal(response)
			c.send <- data
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// BroadcastPriceUpdate 广播价格更新
func BroadcastPriceUpdate(hub *Hub, itemID uint, price float64, platform string) {
	message := Message{
		Type: "price_update",
		Data: map[string]interface{}{
			"item_id":  itemID,
			"price":    price,
			"platform": platform,
			"time":     time.Now(),
		},
	}

	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	hub.broadcast <- data
}

// BroadcastOrderUpdate 广播订单更新
func BroadcastOrderUpdate(hub *Hub, orderType string, order interface{}) {
	message := Message{
		Type: "order_update",
		Data: map[string]interface{}{
			"type":  orderType,
			"order": order,
			"time":  time.Now(),
		},
	}

	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	hub.broadcast <- data
}

// BroadcastNotification 广播通知
func BroadcastNotification(hub *Hub, notification interface{}) {
	message := Message{
		Type: "notification",
		Data: notification,
	}

	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	hub.broadcast <- data
}