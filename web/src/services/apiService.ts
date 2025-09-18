import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || '/api/v1';

// Configure axios defaults
axios.defaults.baseURL = API_BASE_URL;
axios.defaults.timeout = 30000;

// Add request interceptor to include auth token
axios.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Add response interceptor to handle errors
axios.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('auth_token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export interface Item {
  id: number;
  name: string;
  market_name: string;
  icon_url: string;
  type: string;
  weapon: string;
  exterior: string;
  rarity: string;
  collection: string;
  created_at: string;
  updated_at: string;
}

export interface Price {
  id: number;
  item_id: number;
  platform: string;
  price: number;
  volume: number;
  currency: string;
  timestamp: string;
  created_at: string;
}

export interface PriceChart {
  item_name: string;
  data: Array<{
    time: string;
    price: number;
    volume: number;
    platform: string;
  }>;
}

export interface Trade {
  id: number;
  user_id: number;
  item_id: number;
  item: Item;
  platform: string;
  type: string;
  price: number;
  quantity: number;
  status: string;
  trade_id: string;
  created_at: string;
  updated_at: string;
}

export interface Strategy {
  id: number;
  user_id: number;
  name: string;
  description: string;
  item_id: number;
  item: Item;
  buy_price: number;
  sell_price: number;
  max_quantity: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ArbitrageOpportunity {
  item: Item;
  buy_platform: string;
  sell_platform: string;
  buy_price: number;
  sell_price: number;
  profit: number;
  profit_percent: number;
}

export interface MarketTrend {
  id: number;
  item_id: number;
  item: Item;
  platform: string;
  trend_direction: string;
  price_change: number;
  volume_change: number;
  confidence: number;
  analysis_date: string;
  created_at: string;
}

class ApiService {
  // Market API
  async getMarketItems(params: {
    page?: number;
    limit?: number;
    search?: string;
    platform?: string;
  } = {}): Promise<{ items: Item[]; page: number; limit: number }> {
    const response = await axios.get('/market/items', { params });
    return response.data;
  }

  async getItem(id: number): Promise<Item> {
    const response = await axios.get(`/market/items/${id}`);
    return response.data.item;
  }

  async getItemPrices(id: number): Promise<{ [platform: string]: Price }> {
    const response = await axios.get(`/market/items/${id}/prices`);
    return response.data.prices;
  }

  async getPriceChart(id: number, days: number = 7): Promise<PriceChart> {
    const response = await axios.get(`/market/items/${id}/chart`, {
      params: { days }
    });
    return response.data.chart;
  }

  async getItemTrend(id: number, platform: string = 'steam', days: number = 7): Promise<MarketTrend> {
    const response = await axios.get(`/market/items/${id}/trend`, {
      params: { platform, days }
    });
    return response.data.trend;
  }

  async getArbitrageOpportunities(minProfit: number = 10): Promise<ArbitrageOpportunity[]> {
    const response = await axios.get('/market/arbitrage', {
      params: { min_profit: minProfit }
    });
    return response.data.opportunities;
  }

  async getTopMovers(limit: number = 10): Promise<MarketTrend[]> {
    const response = await axios.get('/market/movers', {
      params: { limit }
    });
    return response.data.movers;
  }

  // Trading API
  async getStrategies(userId?: number): Promise<Strategy[]> {
    const response = await axios.get('/trading/strategies', {
      params: userId ? { user_id: userId } : {}
    });
    return response.data.strategies;
  }

  async createStrategy(strategy: Partial<Strategy>): Promise<Strategy> {
    const response = await axios.post('/trading/strategies', strategy);
    return response.data.strategy;
  }

  async updateStrategy(id: number, updates: Partial<Strategy>): Promise<void> {
    await axios.put(`/trading/strategies/${id}`, updates);
  }

  async deleteStrategy(id: number): Promise<void> {
    await axios.delete(`/trading/strategies/${id}`);
  }

  async executeStrategy(id: number): Promise<void> {
    await axios.post(`/trading/strategies/${id}/execute`);
  }

  async getTrades(userId?: number, limit: number = 50): Promise<Trade[]> {
    const response = await axios.get('/trading/trades', {
      params: { user_id: userId, limit }
    });
    return response.data.trades;
  }

  async buyItem(itemId: number, platform: string, price: number): Promise<void> {
    await axios.post('/trading/buy', {
      item_id: itemId,
      platform,
      price
    });
  }

  async sellItem(assetId: string, platform: string, price: number): Promise<void> {
    await axios.post('/trading/sell', {
      asset_id: assetId,
      platform,
      price
    });
  }

  // Inventory API
  async getSteamInventory(steamId: string): Promise<any[]> {
    const response = await axios.get(`/inventory/steam/${steamId}`);
    return response.data.inventory;
  }

  async getBuffInventory(userId: string): Promise<any[]> {
    const response = await axios.get(`/inventory/buff/${userId}`);
    return response.data.inventory;
  }

  async getYoupinInventory(userId: string): Promise<any[]> {
    const response = await axios.get(`/inventory/youpin/${userId}`);
    return response.data.inventory;
  }

  // Analytics API
  async getDashboard(): Promise<{
    recent_trades: Trade[];
    opportunities: ArbitrageOpportunity[];
    top_movers: MarketTrend[];
    timestamp: string;
  }> {
    const response = await axios.get('/analytics/dashboard');
    return response.data;
  }

  async getPerformance(userId?: number): Promise<{
    total_profit: number;
    total_trades: number;
    success_rate: number;
    roi: number;
  }> {
    const response = await axios.get('/analytics/performance', {
      params: userId ? { user_id: userId } : {}
    });
    return response.data;
  }
}

export const apiService = new ApiService();