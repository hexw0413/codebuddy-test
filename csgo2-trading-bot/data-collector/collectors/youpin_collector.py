"""
悠悠有品市场数据采集器
"""

import asyncio
import json
import hashlib
import time
from typing import List, Dict, Any, Optional
from datetime import datetime

from collectors.base_collector import BaseCollector
from utils.logger import setup_logger

logger = setup_logger(__name__)


class YouPinCollector(BaseCollector):
    """悠悠有品市场数据采集器"""
    
    def __init__(self, config, db_manager):
        super().__init__(config, db_manager)
        self.base_url = self.config.YOUPIN_BASE_URL
        
    async def initialize(self):
        """初始化采集器"""
        await super().initialize()
        logger.info("悠悠有品采集器初始化完成")
        
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
                    logger.error(f"采集悠悠有品物品数据失败: {result}")
                    
        except Exception as e:
            logger.error(f"悠悠有品市场数据采集失败: {e}")
            
        return items
        
    async def _get_item_list(self) -> List[Dict[str, Any]]:
        """获取物品列表"""
        items = []
        
        try:
            url = f"{self.base_url}/api/v2/goods/list"
            
            params = {
                'game_id': 730,  # CS2
                'page': 1,
                'page_size': 100,
                'sort_type': 'price_desc',
                'timestamp': int(time.time() * 1000)
            }
            
            # 添加签名
            params['sign'] = self._sign_request(params)
            
            headers = self._get_headers()
            
            response = await self.request_with_retry('GET', url, params=params, headers=headers)
            if response:
                data = await response.json()
                
                if data.get('code') == 0:
                    items = data.get('data', {}).get('list', [])
                    
        except Exception as e:
            logger.error(f"获取悠悠有品物品列表失败: {e}")
            
        return items
        
    async def _collect_item_data(self, item_info: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """采集单个物品数据"""
        try:
            goods_id = item_info.get('goods_id')
            
            # 获取物品详细信息
            item_details = await self._get_item_details(goods_id)
            
            if item_details:
                return {
                    'market_hash_name': item_info.get('market_hash_name', ''),
                    'name': item_info.get('name', ''),
                    'name_cn': item_info.get('name_cn', ''),
                    'type': item_info.get('type', ''),
                    'rarity': item_info.get('rarity', ''),
                    'quality': item_info.get('quality', ''),
                    'icon_url': item_info.get('icon_url', ''),
                    'current_price': float(item_details.get('min_price', 0)),
                    'max_price': float(item_details.get('max_price', 0)),
                    'volume': item_details.get('sell_count', 0),
                    'platform': 'youpin',
                    'goods_id': goods_id,
                    'timestamp': datetime.now()
                }
                
        except Exception as e:
            logger.error(f"采集悠悠有品物品 {item_info.get('name', '')} 数据失败: {e}")
            
        return None
        
    async def _get_item_details(self, goods_id: str) -> Optional[Dict[str, Any]]:
        """获取物品详细信息"""
        try:
            url = f"{self.base_url}/api/v2/goods/detail"
            
            params = {
                'goods_id': goods_id,
                'game_id': 730,
                'timestamp': int(time.time() * 1000)
            }
            
            # 添加签名
            params['sign'] = self._sign_request(params)
            
            headers = self._get_headers()
            
            response = await self.request_with_retry('GET', url, params=params, headers=headers)
            if response:
                data = await response.json()
                
                if data.get('code') == 0:
                    return data.get('data', {})
                    
        except Exception as e:
            logger.error(f"获取悠悠有品物品 {goods_id} 详情失败: {e}")
            
        return None
        
    async def _get_price_history(self, goods_id: str, days: int = 30) -> List[Dict[str, Any]]:
        """获取价格历史"""
        history = []
        
        try:
            url = f"{self.base_url}/api/v2/goods/price_history"
            
            params = {
                'goods_id': goods_id,
                'game_id': 730,
                'days': days,
                'timestamp': int(time.time() * 1000)
            }
            
            # 添加签名
            params['sign'] = self._sign_request(params)
            
            headers = self._get_headers()
            
            response = await self.request_with_retry('GET', url, params=params, headers=headers)
            if response:
                data = await response.json()
                
                if data.get('code') == 0:
                    price_list = data.get('data', {}).get('list', [])
                    
                    for item in price_list:
                        history.append({
                            'date': datetime.fromtimestamp(item.get('timestamp', 0) / 1000),
                            'price': float(item.get('price', 0)),
                            'volume': int(item.get('volume', 0))
                        })
                        
        except Exception as e:
            logger.error(f"获取悠悠有品物品 {goods_id} 价格历史失败: {e}")
            
        return history
        
    def _get_headers(self) -> Dict[str, str]:
        """获取请求头"""
        headers = {
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
            'Accept': 'application/json',
            'Accept-Language': 'zh-CN,zh;q=0.9',
            'Referer': self.base_url,
            'X-API-Key': self.config.YOUPIN_API_KEY
        }
        
        return headers
        
    def _sign_request(self, params: Dict[str, Any]) -> str:
        """签名请求参数"""
        # 排序参数
        sorted_params = sorted(params.items())
        
        # 构建签名字符串
        sign_str = '&'.join([f"{k}={v}" for k, v in sorted_params])
        sign_str += f"&key={self.config.YOUPIN_API_SECRET}"
        
        # 计算MD5
        return hashlib.md5(sign_str.encode()).hexdigest().upper()
        
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
                'platform': 'youpin',
                'timestamp': item_data['timestamp']
            })
            
        except Exception as e:
            logger.error(f"保存悠悠有品物品数据失败: {e}")
            
    async def search_items(self, keyword: str) -> List[Dict[str, Any]]:
        """搜索物品"""
        items = []
        
        try:
            url = f"{self.base_url}/api/v2/goods/search"
            
            params = {
                'game_id': 730,
                'keyword': keyword,
                'page': 1,
                'page_size': 50,
                'timestamp': int(time.time() * 1000)
            }
            
            # 添加签名
            params['sign'] = self._sign_request(params)
            
            headers = self._get_headers()
            
            response = await self.request_with_retry('GET', url, params=params, headers=headers)
            if response:
                data = await response.json()
                
                if data.get('code') == 0:
                    items = data.get('data', {}).get('list', [])
                    
        except Exception as e:
            logger.error(f"搜索悠悠有品物品失败: {e}")
            
        return items
        
    async def close(self):
        """关闭采集器"""
        await super().close()
        logger.info("悠悠有品采集器已关闭")