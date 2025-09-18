import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || '/api/v1';

export interface User {
  id: number;
  steam_id: string;
  username: string;
  avatar: string;
  created_at: string;
  updated_at: string;
}

export interface LoginResponse {
  login_url: string;
}

class AuthService {
  private baseURL: string;

  constructor() {
    this.baseURL = `${API_BASE_URL}/auth`;
  }

  async getSteamLoginUrl(returnUrl?: string): Promise<LoginResponse> {
    const response = await axios.get(`${this.baseURL}/steam/login`, {
      params: { return_url: returnUrl }
    });
    return response.data;
  }

  async handleSteamCallback(params: URLSearchParams): Promise<{ user: User; message: string }> {
    const response = await axios.get(`${this.baseURL}/steam/callback?${params.toString()}`);
    return response.data;
  }

  async getCurrentUser(): Promise<User> {
    const response = await axios.get(`${this.baseURL}/me`);
    return response.data.user;
  }

  async logout(): Promise<void> {
    await axios.post(`${this.baseURL}/logout`);
  }

  isAuthenticated(): boolean {
    // Check if user has valid session/token
    return localStorage.getItem('auth_token') !== null;
  }

  setAuthToken(token: string): void {
    localStorage.setItem('auth_token', token);
    axios.defaults.headers.common['Authorization'] = `Bearer ${token}`;
  }

  removeAuthToken(): void {
    localStorage.removeItem('auth_token');
    delete axios.defaults.headers.common['Authorization'];
  }

  getAuthToken(): string | null {
    return localStorage.getItem('auth_token');
  }
}

export const authService = new AuthService();