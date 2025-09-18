from abc import ABC, abstractmethod
from typing import Dict, List, Optional, Any
import aiohttp
import asyncio
import logging
from datetime import datetime
import time

class BaseCollector(ABC):
    """Base class for all data collectors"""
    
    def __init__(self, api_key: Optional[str] = None):
        self.api_key = api_key
        self.logger = logging.getLogger(self.__class__.__name__)
        self.session: Optional[aiohttp.ClientSession] = None
        self.rate_limit_delay = 2.0  # seconds between requests
        self.last_request_time = 0
        
    async def __aenter__(self):
        """Async context manager entry"""
        await self._create_session()
        return self
        
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        await self._close_session()
        
    async def _create_session(self):
        """Create aiohttp session"""
        if not self.session:
            timeout = aiohttp.ClientTimeout(total=30)
            connector = aiohttp.TCPConnector(limit=10)
            self.session = aiohttp.ClientSession(
                timeout=timeout,
                connector=connector,
                headers=self._get_default_headers()
            )
            
    async def _close_session(self):
        """Close aiohttp session"""
        if self.session:
            await self.session.close()
            self.session = None
            
    def _get_default_headers(self) -> Dict[str, str]:
        """Get default headers for requests"""
        return {
            'User-Agent': 'CSGO-Trader/1.0 (Data Collector)',
            'Accept': 'application/json',
            'Accept-Language': 'en-US,en;q=0.9'
        }
        
    async def _rate_limit(self):
        """Apply rate limiting"""
        current_time = time.time()
        time_since_last = current_time - self.last_request_time
        
        if time_since_last < self.rate_limit_delay:
            sleep_time = self.rate_limit_delay - time_since_last
            await asyncio.sleep(sleep_time)
            
        self.last_request_time = time.time()
        
    async def _make_request(self, url: str, method: str = 'GET', **kwargs) -> Optional[Dict[str, Any]]:
        """Make HTTP request with error handling and rate limiting"""
        await self._rate_limit()
        
        if not self.session:
            await self._create_session()
            
        try:
            async with self.session.request(method, url, **kwargs) as response:
                if response.status == 200:
                    return await response.json()
                elif response.status == 429:  # Rate limited
                    self.logger.warning(f"Rate limited by {self.get_platform_name()}, waiting...")
                    await asyncio.sleep(60)  # Wait 1 minute
                    return await self._make_request(url, method, **kwargs)
                else:
                    self.logger.error(f"HTTP {response.status} error for {url}")
                    return None
                    
        except asyncio.TimeoutError:
            self.logger.error(f"Timeout error for {url}")
            return None
        except Exception as e:
            self.logger.error(f"Request error for {url}: {e}")
            return None
            
    @abstractmethod
    def get_platform_name(self) -> str:
        """Get platform name"""
        pass
        
    @abstractmethod
    async def get_item_price(self, market_name: str) -> Optional[Dict[str, Any]]:
        """Get price for a specific item"""
        pass
        
    @abstractmethod
    async def get_popular_items(self, limit: int = 100) -> List[Dict[str, Any]]:
        """Get list of popular items"""
        pass
        
    async def get_item_history(self, item_id: str, days: int = 7) -> List[Dict[str, Any]]:
        """Get price history for an item (optional implementation)"""
        self.logger.warning(f"Price history not implemented for {self.get_platform_name()}")
        return []
        
    def _normalize_price_data(self, raw_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Normalize price data to standard format"""
        try:
            return {
                'price': float(raw_data.get('price', 0)),
                'volume': int(raw_data.get('volume', 0)),
                'currency': raw_data.get('currency', 'USD'),
                'timestamp': datetime.now().isoformat(),
                'platform': self.get_platform_name(),
                'raw_data': raw_data
            }
        except (ValueError, TypeError) as e:
            self.logger.error(f"Error normalizing price data: {e}")
            return None
            
    def _normalize_item_data(self, raw_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Normalize item data to standard format"""
        try:
            return {
                'name': raw_data.get('name', ''),
                'market_name': raw_data.get('market_name', ''),
                'icon_url': raw_data.get('icon_url', ''),
                'type': raw_data.get('type', ''),
                'weapon': raw_data.get('weapon', ''),
                'exterior': raw_data.get('exterior', ''),
                'rarity': raw_data.get('rarity', ''),
                'collection': raw_data.get('collection', ''),
                'platform': self.get_platform_name(),
                'raw_data': raw_data
            }
        except Exception as e:
            self.logger.error(f"Error normalizing item data: {e}")
            return None