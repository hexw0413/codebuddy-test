"""
Steam市场数据采集器
"""

import asyncio
import json
import re
from typing import List, Dict, Any, Optional
from datetime import datetime
from urllib.parse import quote

import aiohttp
from bs4 import BeautifulSoup
from fake_useragent import UserAgent

from collectors.base_collector import BaseCollector
from utils.logger import setup_logger

logger = setup_logger(__name__)


class SteamCollector(BaseCollector):
    """Steam市场数据采集器"""
    
    def __init__(self, config, db_manager):
        super().__init__(config, db_manager)
        self.base_url = self.config.STEAM_MARKET_URL
        self.app_id = 730  # CS2的AppID
        self.ua = UserAgent()
        
    async def initialize(self):
        """初始化采集器"""
        await super().initialize()
        logger.info("Steam采集器初始化完成")
        
    async def collect_market_data(self) -> List[Dict[str, Any]]:
        """采集市场数据"""
        items = []
        
        try:
            # 获取热门物品列表
            popular_items = await self._get_popular_items()
            
            # 并发采集物品详情
            tasks = []
            for item_name in popular_items[:self.config.MAX_ITEMS_PER_RUN]:
                tasks.append(self._collect_item_data(item_name))
                
            results = await asyncio.gather(*tasks, return_exceptions=True)
            
            for result in results:
                if isinstance(result, dict):
                    items.append(result)
                    # 保存到数据库
                    await self._save_item_data(result)
                elif isinstance(result, Exception):
                    logger.error(f"采集物品数据失败: {result}")
                    
        except Exception as e:
            logger.error(f"Steam市场数据采集失败: {e}")
            
        return items
        
    async def _get_popular_items(self) -> List[str]:
        """获取热门物品列表"""
        items = []
        
        try:
            url = f"{self.base_url}/search/render/"
            params = {
                'query': '',
                'start': 0,
                'count': 100,
                'search_descriptions': 0,
                'sort_column': 'popular',
                'sort_dir': 'desc',
                'appid': self.app_id,
                'norender': 1
            }
            
            headers = {
                'User-Agent': self.ua.random,
                'Accept': 'application/json',
                'Referer': self.base_url
            }
            
            async with self.session.get(url, params=params, headers=headers) as response:
                if response.status == 200:
                    data = await response.json()
                    
                    if data.get('success'):
                        results = data.get('results', [])
                        for item in results:
                            items.append(item.get('hash_name'))
                            
        except Exception as e:
            logger.error(f"获取热门物品列表失败: {e}")
            
        return items
        
    async def _collect_item_data(self, item_name: str) -> Optional[Dict[str, Any]]:
        """采集单个物品数据"""
        try:
            # 获取物品价格信息
            price_data = await self._get_item_price(item_name)
            
            # 获取物品详细信息
            item_details = await self._get_item_details(item_name)
            
            if price_data and item_details:
                return {
                    'market_hash_name': item_name,
                    'name': item_details.get('name', item_name),
                    'type': item_details.get('type', ''),
                    'rarity': item_details.get('rarity', ''),
                    'quality': item_details.get('quality', ''),
                    'icon_url': item_details.get('icon_url', ''),
                    'current_price': price_data.get('lowest_price', 0),
                    'median_price': price_data.get('median_price', 0),
                    'volume': price_data.get('volume', 0),
                    'platform': 'steam',
                    'timestamp': datetime.now()
                }
                
        except Exception as e:
            logger.error(f"采集物品 {item_name} 数据失败: {e}")
            
        return None
        
    async def _get_item_price(self, item_name: str) -> Optional[Dict[str, Any]]:
        """获取物品价格信息"""
        try:
            url = f"{self.base_url}/priceoverview/"
            params = {
                'appid': self.app_id,
                'currency': 23,  # CNY
                'market_hash_name': item_name
            }
            
            headers = {
                'User-Agent': self.ua.random,
                'Accept': 'application/json'
            }
            
            async with self.session.get(url, params=params, headers=headers) as response:
                if response.status == 200:
                    data = await response.json()
                    
                    if data.get('success'):
                        return {
                            'lowest_price': self._parse_price(data.get('lowest_price', '0')),
                            'median_price': self._parse_price(data.get('median_price', '0')),
                            'volume': int(data.get('volume', '0').replace(',', ''))
                        }
                        
        except Exception as e:
            logger.error(f"获取物品 {item_name} 价格失败: {e}")
            
        return None
        
    async def _get_item_details(self, item_name: str) -> Optional[Dict[str, Any]]:
        """获取物品详细信息"""
        try:
            url = f"{self.base_url}/listings/{self.app_id}/{quote(item_name)}"
            
            headers = {
                'User-Agent': self.ua.random,
                'Accept': 'text/html,application/xhtml+xml'
            }
            
            async with self.session.get(url, headers=headers) as response:
                if response.status == 200:
                    html = await response.text()
                    soup = BeautifulSoup(html, 'lxml')
                    
                    # 解析物品信息
                    details = {}
                    
                    # 获取物品图标
                    icon_elem = soup.find('img', class_='market_listing_largeimage')
                    if icon_elem:
                        details['icon_url'] = icon_elem.get('src', '')
                    
                    # 解析物品类型和稀有度
                    item_info = soup.find('div', class_='market_listing_nav')
                    if item_info:
                        info_text = item_info.get_text(strip=True)
                        details['type'] = self._extract_item_type(info_text)
                        details['rarity'] = self._extract_item_rarity(info_text)
                        details['quality'] = self._extract_item_quality(info_text)
                    
                    details['name'] = item_name
                    return details
                    
        except Exception as e:
            logger.error(f"获取物品 {item_name} 详情失败: {e}")
            
        return None
        
    async def _get_price_history(self, item_name: str, days: int = 30) -> List[Dict[str, Any]]:
        """获取价格历史"""
        history = []
        
        try:
            url = f"{self.base_url}/pricehistory/"
            params = {
                'appid': self.app_id,
                'market_hash_name': item_name
            }
            
            headers = {
                'User-Agent': self.ua.random,
                'Accept': 'application/json',
                'Cookie': self._get_steam_cookies()
            }
            
            async with self.session.get(url, params=params, headers=headers) as response:
                if response.status == 200:
                    data = await response.json()
                    
                    if data.get('success'):
                        prices = data.get('prices', [])
                        
                        for price_point in prices[-days*24:]:  # 获取最近N天的数据
                            timestamp, price, volume = price_point
                            history.append({
                                'timestamp': datetime.strptime(timestamp, '%b %d %Y %H: +0'),
                                'price': float(price),
                                'volume': int(volume)
                            })
                            
        except Exception as e:
            logger.error(f"获取物品 {item_name} 价格历史失败: {e}")
            
        return history
        
    async def _save_item_data(self, item_data: Dict[str, Any]):
        """保存物品数据到数据库"""
        try:
            # 保存或更新物品信息
            await self.db_manager.upsert_item(item_data)
            
            # 保存价格历史
            await self.db_manager.save_price_history({
                'market_hash_name': item_data['market_hash_name'],
                'price': item_data['current_price'],
                'volume': item_data.get('volume', 0),
                'platform': 'steam',
                'timestamp': item_data['timestamp']
            })
            
        except Exception as e:
            logger.error(f"保存物品数据失败: {e}")
            
    def _parse_price(self, price_str: str) -> float:
        """解析价格字符串"""
        try:
            # 移除货币符号和空格
            price_str = re.sub(r'[^\d.,]', '', price_str)
            # 将逗号替换为点（处理不同地区的格式）
            price_str = price_str.replace(',', '.')
            return float(price_str)
        except:
            return 0.0
            
    def _extract_item_type(self, text: str) -> str:
        """提取物品类型"""
        patterns = [
            r'(Rifle|Pistol|Knife|Gloves|SMG|Sniper Rifle|Shotgun|Machine Gun)',
            r'(Sticker|Graffiti|Music Kit|Case|Key|Pass|Pin)',
            r'(Agent|Patch)'
        ]
        
        for pattern in patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                return match.group(1)
                
        return 'Other'
        
    def _extract_item_rarity(self, text: str) -> str:
        """提取物品稀有度"""
        rarities = [
            'Contraband',
            'Covert',
            'Classified',
            'Restricted',
            'Mil-Spec',
            'Industrial Grade',
            'Consumer Grade'
        ]
        
        for rarity in rarities:
            if rarity.lower() in text.lower():
                return rarity
                
        return 'Common'
        
    def _extract_item_quality(self, text: str) -> str:
        """提取物品品质"""
        qualities = [
            'Factory New',
            'Minimal Wear',
            'Field-Tested',
            'Well-Worn',
            'Battle-Scarred'
        ]
        
        for quality in qualities:
            if quality.lower() in text.lower():
                return quality
                
        return 'Not Applicable'
        
    def _get_steam_cookies(self) -> str:
        """获取Steam cookies（如果需要）"""
        # 这里可以实现Steam登录后的cookie管理
        return ""
        
    async def close(self):
        """关闭采集器"""
        await super().close()
        logger.info("Steam采集器已关闭")