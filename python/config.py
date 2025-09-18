import os
from typing import Optional
from dotenv import load_dotenv

load_dotenv()

class Config:
    """Configuration class for the Python data collection service"""
    
    def __init__(self):
        # Database
        self.DATABASE_URL: str = os.getenv('DATABASE_URL', 'sqlite:///csgo_trader.db')
        
        # API Keys
        self.STEAM_API_KEY: Optional[str] = os.getenv('STEAM_API_KEY')
        self.BUFF_API_KEY: Optional[str] = os.getenv('BUFF_API_KEY')
        self.YOUPIN_API_KEY: Optional[str] = os.getenv('YOUPIN_API_KEY')
        
        # Collection settings
        self.PRICE_COLLECTION_INTERVAL: int = int(os.getenv('PRICE_COLLECTION_INTERVAL', '15'))  # minutes
        self.MARKET_DATA_INTERVAL: int = int(os.getenv('MARKET_DATA_INTERVAL', '30'))  # minutes
        self.TREND_ANALYSIS_INTERVAL: int = int(os.getenv('TREND_ANALYSIS_INTERVAL', '60'))  # minutes
        
        # Rate limiting
        self.REQUEST_DELAY: float = float(os.getenv('REQUEST_DELAY', '2.0'))  # seconds
        self.MAX_RETRIES: int = int(os.getenv('MAX_RETRIES', '3'))
        
        # Data retention
        self.PRICE_DATA_RETENTION_DAYS: int = int(os.getenv('PRICE_DATA_RETENTION_DAYS', '30'))
        
        # Logging
        self.LOG_LEVEL: str = os.getenv('LOG_LEVEL', 'INFO')
        self.LOG_FILE: Optional[str] = os.getenv('LOG_FILE', 'logs/data_collector.log')
        
        # Proxy settings (if needed)
        self.PROXY_HTTP: Optional[str] = os.getenv('PROXY_HTTP')
        self.PROXY_HTTPS: Optional[str] = os.getenv('PROXY_HTTPS')
        
    def get_proxy_dict(self) -> Optional[dict]:
        """Get proxy configuration dictionary"""
        if self.PROXY_HTTP or self.PROXY_HTTPS:
            return {
                'http': self.PROXY_HTTP,
                'https': self.PROXY_HTTPS
            }
        return None
        
    def validate(self) -> bool:
        """Validate configuration"""
        if not self.STEAM_API_KEY:
            print("Warning: STEAM_API_KEY not set")
            
        if not self.BUFF_API_KEY:
            print("Warning: BUFF_API_KEY not set")
            
        if not self.YOUPIN_API_KEY:
            print("Warning: YOUPIN_API_KEY not set")
            
        return True