"""
配置管理模块
"""

import os
from typing import Optional
from dotenv import load_dotenv

# 加载环境变量
load_dotenv()


class Config:
    """配置类"""
    
    def __init__(self):
        # 数据库配置
        self.DB_HOST = os.getenv('DB_HOST', 'localhost')
        self.DB_PORT = int(os.getenv('DB_PORT', 5432))
        self.DB_NAME = os.getenv('DB_NAME', 'csgo2_trading')
        self.DB_USER = os.getenv('DB_USER', 'postgres')
        self.DB_PASSWORD = os.getenv('DB_PASSWORD', '')
        
        # Redis配置
        self.REDIS_HOST = os.getenv('REDIS_HOST', 'localhost')
        self.REDIS_PORT = int(os.getenv('REDIS_PORT', 6379))
        self.REDIS_DB = int(os.getenv('REDIS_DB', 0))
        self.REDIS_PASSWORD = os.getenv('REDIS_PASSWORD', '')
        
        # Steam配置
        self.STEAM_ENABLED = os.getenv('STEAM_ENABLED', 'true').lower() == 'true'
        self.STEAM_API_KEY = os.getenv('STEAM_API_KEY', '')
        self.STEAM_MARKET_URL = 'https://steamcommunity.com/market'
        self.STEAM_RATE_LIMIT = int(os.getenv('STEAM_RATE_LIMIT', 20))  # 每分钟请求数
        
        # BUFF配置
        self.BUFF_ENABLED = os.getenv('BUFF_ENABLED', 'true').lower() == 'true'
        self.BUFF_BASE_URL = 'https://buff.163.com'
        self.BUFF_APP_ID = os.getenv('BUFF_APP_ID', '')
        self.BUFF_APP_SECRET = os.getenv('BUFF_APP_SECRET', '')
        self.BUFF_COOKIE = os.getenv('BUFF_COOKIE', '')
        self.BUFF_RATE_LIMIT = int(os.getenv('BUFF_RATE_LIMIT', 30))
        
        # 悠悠有品配置
        self.YOUPIN_ENABLED = os.getenv('YOUPIN_ENABLED', 'true').lower() == 'true'
        self.YOUPIN_BASE_URL = 'https://www.youpin898.com'
        self.YOUPIN_API_KEY = os.getenv('YOUPIN_API_KEY', '')
        self.YOUPIN_API_SECRET = os.getenv('YOUPIN_API_SECRET', '')
        self.YOUPIN_RATE_LIMIT = int(os.getenv('YOUPIN_RATE_LIMIT', 30))
        
        # 采集配置
        self.COLLECT_INTERVAL = int(os.getenv('COLLECT_INTERVAL', 300))  # 秒
        self.MAX_ITEMS_PER_RUN = int(os.getenv('MAX_ITEMS_PER_RUN', 100))
        self.CONCURRENT_REQUESTS = int(os.getenv('CONCURRENT_REQUESTS', 5))
        self.REQUEST_TIMEOUT = int(os.getenv('REQUEST_TIMEOUT', 30))
        self.RETRY_TIMES = int(os.getenv('RETRY_TIMES', 3))
        self.RETRY_DELAY = int(os.getenv('RETRY_DELAY', 5))
        
        # 代理配置
        self.USE_PROXY = os.getenv('USE_PROXY', 'false').lower() == 'true'
        self.PROXY_URL = os.getenv('PROXY_URL', '')
        self.PROXY_USER = os.getenv('PROXY_USER', '')
        self.PROXY_PASSWORD = os.getenv('PROXY_PASSWORD', '')
        
        # 套利配置
        self.MIN_ARBITRAGE_PROFIT = float(os.getenv('MIN_ARBITRAGE_PROFIT', 5.0))  # 最小套利利润率(%)
        
        # 日志配置
        self.LOG_LEVEL = os.getenv('LOG_LEVEL', 'INFO')
        self.LOG_FILE = os.getenv('LOG_FILE', 'data_collector.log')
        self.LOG_MAX_SIZE = int(os.getenv('LOG_MAX_SIZE', 10))  # MB
        self.LOG_BACKUP_COUNT = int(os.getenv('LOG_BACKUP_COUNT', 5))
        
        # 通知配置
        self.ENABLE_NOTIFICATIONS = os.getenv('ENABLE_NOTIFICATIONS', 'false').lower() == 'true'
        self.WEBHOOK_URL = os.getenv('WEBHOOK_URL', '')
        
    def get_database_url(self) -> str:
        """获取数据库连接URL"""
        return f"postgresql://{self.DB_USER}:{self.DB_PASSWORD}@{self.DB_HOST}:{self.DB_PORT}/{self.DB_NAME}"
        
    def get_redis_url(self) -> str:
        """获取Redis连接URL"""
        if self.REDIS_PASSWORD:
            return f"redis://:{self.REDIS_PASSWORD}@{self.REDIS_HOST}:{self.REDIS_PORT}/{self.REDIS_DB}"
        return f"redis://{self.REDIS_HOST}:{self.REDIS_PORT}/{self.REDIS_DB}"
        
    def get_proxy(self) -> Optional[dict]:
        """获取代理配置"""
        if not self.USE_PROXY:
            return None
            
        proxy = {
            'http': self.PROXY_URL,
            'https': self.PROXY_URL
        }
        
        if self.PROXY_USER and self.PROXY_PASSWORD:
            auth = f"{self.PROXY_USER}:{self.PROXY_PASSWORD}@"
            proxy['http'] = proxy['http'].replace('://', f'://{auth}')
            proxy['https'] = proxy['https'].replace('://', f'://{auth}')
            
        return proxy