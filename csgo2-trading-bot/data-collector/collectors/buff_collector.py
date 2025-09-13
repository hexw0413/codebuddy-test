"""
BUFF市场数据采集器
"""

import asyncio
import json
import hashlib
import time
from typing import List, Dict, Any, Optional
from datetime import datetime
from urllib.parse import urlencode

from collectors.base_collector import BaseCollector
from utils.logger import setup_logger

logger = setup_logger(__name__)


class BuffCollector(BaseCollector):
    """BUFF市场数据采集器"""
    
    def __init__(self, config, db_manager):
        super().__init__(config, db_manager)
        self.base_url = self.config.BUFF_BASE_URL
        self.game_id = 730  # CS2
        
    async def initialize(self):
        """初始化采集器"""
        await super().initialize()
        logger.info("BUFF采集器初始化完成")
        
    async def collect_market_data(self) -> List[Dict[str, Any]]:
        """采集市场数据"""
        items = []
        
        try:
            # 获取物品列表
            item_list = await self._get_item_list()
            
            # 并发采集物品详情
            tasks = []
            for item in item_list[:self.config.MAX_ITEMS_PER_RUN]:
                tasks.append(self._collect_item_data(item))
                
            results = await asyncio.gather(*tasks, return_exceptions=True)
            
            for result in results:
                if isinstance(result, dict):
                    items.append(result)
                    # 保存到数据库
                    await self._save_item_data(result)
                elif isinstance(result, Exception):
                    logger.error(f"采集BUFF物品数据失败: {result}")
                    
        except Exception as e:
            logger.error(f"BUFF市场数据采集失败: {e}")
            
        return items
        
    async def _get_item_list(self) -> List[Dict[str, Any]]:
        """获取物品列表"""
        items = []
        
        try:
            url = f"{self.base_url}/api/market/goods"
            
            params = {
                'game': 'csgo',
                'page_num': 1,
                'page_size': 100,
                'sort_by': 'price.desc',
                'min_price': 1,
                'max_price': 100000,
                '_': int(time.time() * 1000)
            }
            
            headers = self._get_headers()
            
            async with self.session.get(url, params=params, headers=headers) as response:
                if response.status == 200:
                    data = await response.json()
                    
                    if data.get('code') == 'OK':
                        items = data.get('data', {}).get('items', [])
                        
        except Exception as e:
            logger.error(f"获取BUFF物品列表失败: {e}")
            
        return items
        
    async def _collect_item_data(self, item_info: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """采集单个物品数据"""
        try:
            goods_id = item_info.get('id')
            
            # 获取物品详细信息
            item_details = await self._get_item_details(goods_id)
            
            if item_details:
                return {
                    'market_hash_name': item_info.get('market_hash_name', ''),
                    'name': item_info.get('name', ''),
                    'name_cn': item_info.get('short_name', ''),
                    'type': item_info.get('goods_info', {}).get('info', {}).get('tags', {}).get('type', {}).get('localized_name', ''),
                    'rarity': item_info.get('goods_info', {}).get('info', {}).get('tags', {}).get('rarity', {}).get('localized_name', ''),
                    'quality': item_info.get('goods_info', {}).get('info', {}).get('tags', {}).get('quality', {}).get('localized_name', ''),
                    'icon_url': item_info.get('goods_info', {}).get('icon_url', ''),
                    'current_price': float(item_details.get('sell_min_price', 0)),
                    'buy_max_price': float(item_details.get('buy_max_price', 0)),
                    'volume': item_details.get('sell_num', 0),
                    'platform': 'buff',
                    'goods_id': goods_id,
                    'steam_price': float(item_info.get('goods_info', {}).get('steam_price_cny', 0)),
                    'timestamp': datetime.now()
                }
                
        except Exception as e:
            logger.error(f"采集BUFF物品 {item_info.get('name', '')} 数据失败: {e}")
            
        return None
        
    async def _get_item_details(self, goods_id: str) -> Optional[Dict[str, Any]]:
        """获取物品详细信息"""
        try:
            url = f"{self.base_url}/api/market/goods/sell_order"
            
            params = {
                'game': 'csgo',
                'goods_id': goods_id,
                'page_num': 1,
                'sort_by': 'default',
                'mode': '',
                'allow_tradable_cooldown': 1,
                '_': int(time.time() * 1000)
            }
            
            headers = self._get_headers()
            
            async with self.session.get(url, params=params, headers=headers) as response:
                if response.status == 200:
                    data = await response.json()
                    
                    if data.get('code') == 'OK':
                        return data.get('data', {})
                        
        except Exception as e:
            logger.error(f"获取BUFF物品 {goods_id} 详情失败: {e}")
            
        return None
        
    async def _get_price_history(self, goods_id: str, days: int = 30) -> List[Dict[str, Any]]:
        """获取价格历史"""
        history = []
        
        try:
            url = f"{self.base_url}/api/market/goods/price_history"
            
            params = {
                'game': 'csgo',
                'goods_id': goods_id,
                'days': days,
                'buff_price_type': 2,  # 求购价格
                '_': int(time.time() * 1000)
            }
            
            headers = self._get_headers()
            
            async with self.session.get(url, params=params, headers=headers) as response:
                if response.status == 200:
                    data = await response.json()
                    
                    if data.get('code') == 'OK':
                        price_history = data.get('data', {}).get('price_history', [])
                        
                        for date_str, price_info in price_history:
                            history.append({
                                'date': datetime.strptime(date_str, '%Y-%m-%d'),
                                'price': float(price_info[0]),
                                'volume': int(price_info[1])
                            })
                            
        except Exception as e:
            logger.error(f"获取BUFF物品 {goods_id} 价格历史失败: {e}")
            
        return history
        
    async def _get_bill_order(self, goods_id: str) -> Optional[Dict[str, Any]]:
        """获取交易订单信息"""
        try:
            url = f"{self.base_url}/api/market/goods/bill_order"
            
            params = {
                'game': 'csgo',
                'goods_id': goods_id,
                '_': int(time.time() * 1000)
            }
            
            headers = self._get_headers()
            
            async with self.session.get(url, params=params, headers=headers) as response:
                if response.status == 200:
                    data = await response.json()
                    
                    if data.get('code') == 'OK':
                        return data.get('data', {})
                        
        except Exception as e:
            logger.error(f"获取BUFF物品 {goods_id} 订单信息失败: {e}")
            
        return None
        
    def _get_headers(self) -> Dict[str, str]:
        """获取请求头"""
        headers = {
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
            'Accept': 'application/json, text/javascript, */*; q=0.01',
            'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
            'X-Requested-With': 'XMLHttpRequest',
            'Referer': f'{self.base_url}/market/csgo',
            'Cookie': self.config.BUFF_COOKIE
        }
        
        return headers
        
    def _sign_request(self, params: Dict[str, Any]) -> str:
        """签名请求参数"""
        # 排序参数
        sorted_params = sorted(params.items())
        
        # 构建签名字符串
        sign_str = urlencode(sorted_params) + self.config.BUFF_APP_SECRET
        
        # 计算MD5
        return hashlib.md5(sign_str.encode()).hexdigest()
        
    async def _save_item_data(self, item_data: Dict[str, Any]):
        """保存物品数据到数据库"""
        try:
            # 保存或更新物品信息
            await self.db_manager.upsert_item(item_data)
            
            # 保存价格历史
            await self.db_manager.save_price_history({
                'market_hash_name': item_data['market_hash_name'],
                'price': item_data['current_price'],
                'buy_price': item_data.get('buy_max_price', 0),
                'volume': item_data.get('volume', 0),
                'platform': 'buff',
                'timestamp': item_data['timestamp']
            })
            
            # 保存跨平台价格对比
            if item_data.get('steam_price'):
                await self.db_manager.save_price_comparison({
                    'market_hash_name': item_data['market_hash_name'],
                    'buff_price': item_data['current_price'],
                    'steam_price': item_data['steam_price'],
                    'price_diff': item_data['steam_price'] - item_data['current_price'],
                    'price_diff_rate': (item_data['steam_price'] - item_data['current_price']) / item_data['current_price'] * 100,
                    'timestamp': item_data['timestamp']
                })
                
        except Exception as e:
            logger.error(f"保存BUFF物品数据失败: {e}")
            
    async def search_items(self, keyword: str) -> List[Dict[str, Any]]:
        """搜索物品"""
        items = []
        
        try:
            url = f"{self.base_url}/api/market/search"
            
            params = {
                'game': 'csgo',
                'text': keyword,
                'page_num': 1,
                'page_size': 50,
                '_': int(time.time() * 1000)
            }
            
            headers = self._get_headers()
            
            async with self.session.get(url, params=params, headers=headers) as response:
                if response.status == 200:
                    data = await response.json()
                    
                    if data.get('code') == 'OK':
                        items = data.get('data', {}).get('items', [])
                        
        except Exception as e:
            logger.error(f"搜索BUFF物品失败: {e}")
            
        return items
        
    async def close(self):
        """关闭采集器"""
        await super().close()
        logger.info("BUFF采集器已关闭")