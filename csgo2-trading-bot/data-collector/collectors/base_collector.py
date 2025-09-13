"""
基础采集器类
"""

import asyncio
from abc import ABC, abstractmethod
from typing import List, Dict, Any, Optional
import aiohttp
from aiohttp import ClientTimeout, TCPConnector

from utils.logger import setup_logger

logger = setup_logger(__name__)


class BaseCollector(ABC):
    """基础采集器抽象类"""
    
    def __init__(self, config, db_manager):
        self.config = config
        self.db_manager = db_manager
        self.session: Optional[aiohttp.ClientSession] = None
        self.rate_limiter = RateLimiter()
        
    async def initialize(self):
        """初始化采集器"""
        # 创建HTTP会话
        timeout = ClientTimeout(total=self.config.REQUEST_TIMEOUT)
        connector = TCPConnector(limit=self.config.CONCURRENT_REQUESTS)
        
        self.session = aiohttp.ClientSession(
            timeout=timeout,
            connector=connector
        )
        
        # 设置代理
        if self.config.USE_PROXY:
            self.session._default_proxy = self.config.get_proxy()
            
    @abstractmethod
    async def collect_market_data(self) -> List[Dict[str, Any]]:
        """采集市场数据（需要子类实现）"""
        pass
        
    async def request_with_retry(self, method: str, url: str, **kwargs) -> Optional[aiohttp.ClientResponse]:
        """带重试的HTTP请求"""
        for attempt in range(self.config.RETRY_TIMES):
            try:
                # 速率限制
                await self.rate_limiter.acquire()
                
                async with self.session.request(method, url, **kwargs) as response:
                    if response.status == 200:
                        return response
                    elif response.status == 429:  # Too Many Requests
                        wait_time = int(response.headers.get('Retry-After', 60))
                        logger.warning(f"触发速率限制，等待 {wait_time} 秒")
                        await asyncio.sleep(wait_time)
                    else:
                        logger.warning(f"请求失败，状态码: {response.status}")
                        
            except asyncio.TimeoutError:
                logger.warning(f"请求超时: {url}")
            except Exception as e:
                logger.error(f"请求异常: {e}")
                
            if attempt < self.config.RETRY_TIMES - 1:
                await asyncio.sleep(self.config.RETRY_DELAY * (attempt + 1))
                
        return None
        
    async def close(self):
        """关闭采集器"""
        if self.session:
            await self.session.close()
            

class RateLimiter:
    """速率限制器"""
    
    def __init__(self, rate: int = 10, per: int = 60):
        """
        :param rate: 请求数量
        :param per: 时间窗口（秒）
        """
        self.rate = rate
        self.per = per
        self.allowance = rate
        self.last_check = asyncio.get_event_loop().time()
        
    async def acquire(self):
        """获取许可"""
        current = asyncio.get_event_loop().time()
        time_passed = current - self.last_check
        self.last_check = current
        
        self.allowance += time_passed * (self.rate / self.per)
        
        if self.allowance > self.rate:
            self.allowance = self.rate
            
        if self.allowance < 1.0:
            sleep_time = (1.0 - self.allowance) * (self.per / self.rate)
            await asyncio.sleep(sleep_time)
            self.allowance = 0.0
        else:
            self.allowance -= 1.0