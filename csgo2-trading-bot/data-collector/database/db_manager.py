"""
数据库管理模块
"""

import asyncio
from typing import Dict, Any, List, Optional
from datetime import datetime, timedelta
import asyncpg
import redis.asyncio as aioredis
from utils.logger import setup_logger

logger = setup_logger(__name__)


class DatabaseManager:
    """数据库管理器"""
    
    def __init__(self, config):
        self.config = config
        self.pool: Optional[asyncpg.Pool] = None
        self.redis: Optional[aioredis.Redis] = None
        
    async def connect(self):
        """连接数据库"""
        try:
            # 连接PostgreSQL
            self.pool = await asyncpg.create_pool(
                host=self.config.DB_HOST,
                port=self.config.DB_PORT,
                user=self.config.DB_USER,
                password=self.config.DB_PASSWORD,
                database=self.config.DB_NAME,
                min_size=5,
                max_size=20
            )
            logger.info("PostgreSQL连接池创建成功")
            
            # 连接Redis
            self.redis = await aioredis.from_url(
                self.config.get_redis_url(),
                encoding="utf-8",
                decode_responses=True
            )
            logger.info("Redis连接成功")
            
        except Exception as e:
            logger.error(f"数据库连接失败: {e}")
            raise
            
    async def close(self):
        """关闭数据库连接"""
        if self.pool:
            await self.pool.close()
        if self.redis:
            await self.redis.close()
            
    async def upsert_item(self, item_data: Dict[str, Any]):
        """插入或更新物品信息"""
        async with self.pool.acquire() as conn:
            await conn.execute("""
                INSERT INTO items (
                    market_hash_name, name, type, rarity, quality,
                    icon_url, current_price, avg_price_7days, avg_price_30days,
                    volume_24h, last_updated
                ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
                ON CONFLICT (market_hash_name) DO UPDATE SET
                    name = EXCLUDED.name,
                    type = EXCLUDED.type,
                    rarity = EXCLUDED.rarity,
                    quality = EXCLUDED.quality,
                    icon_url = EXCLUDED.icon_url,
                    current_price = EXCLUDED.current_price,
                    volume_24h = EXCLUDED.volume_24h,
                    last_updated = EXCLUDED.last_updated
            """,
                item_data.get('market_hash_name'),
                item_data.get('name'),
                item_data.get('type'),
                item_data.get('rarity'),
                item_data.get('quality'),
                item_data.get('icon_url'),
                item_data.get('current_price', 0),
                item_data.get('avg_price_7days', 0),
                item_data.get('avg_price_30days', 0),
                item_data.get('volume', 0),
                datetime.now()
            )
            
    async def save_price_history(self, price_data: Dict[str, Any]):
        """保存价格历史"""
        async with self.pool.acquire() as conn:
            # 获取物品ID
            item_id = await conn.fetchval(
                "SELECT id FROM items WHERE market_hash_name = $1",
                price_data.get('market_hash_name')
            )
            
            if item_id:
                await conn.execute("""
                    INSERT INTO price_histories (
                        item_id, price, volume, platform, recorded_at
                    ) VALUES ($1, $2, $3, $4, $5)
                """,
                    item_id,
                    price_data.get('price', 0),
                    price_data.get('volume', 0),
                    price_data.get('platform'),
                    price_data.get('timestamp', datetime.now())
                )
                
    async def save_price_comparison(self, comparison_data: Dict[str, Any]):
        """保存价格对比数据"""
        # 缓存到Redis
        key = f"price_comparison:{comparison_data['market_hash_name']}"
        await self.redis.hset(key, mapping=comparison_data)
        await self.redis.expire(key, 3600)  # 1小时过期
        
    async def get_cross_platform_prices(self) -> List[Dict[str, Any]]:
        """获取跨平台价格"""
        async with self.pool.acquire() as conn:
            rows = await conn.fetch("""
                SELECT 
                    i.id,
                    i.market_hash_name as name,
                    i.current_price as steam_price,
                    (
                        SELECT price FROM price_histories 
                        WHERE item_id = i.id AND platform = 'buff'
                        ORDER BY recorded_at DESC LIMIT 1
                    ) as buff_price,
                    (
                        SELECT price FROM price_histories 
                        WHERE item_id = i.id AND platform = 'youpin'
                        ORDER BY recorded_at DESC LIMIT 1
                    ) as youpin_price
                FROM items i
                WHERE i.current_price > 0
            """)
            
            result = []
            for row in rows:
                prices = {}
                if row['steam_price']:
                    prices['steam'] = float(row['steam_price'])
                if row['buff_price']:
                    prices['buff'] = float(row['buff_price'])
                if row['youpin_price']:
                    prices['youpin'] = float(row['youpin_price'])
                    
                if len(prices) >= 2:
                    result.append({
                        'id': row['id'],
                        'name': row['name'],
                        'prices': prices
                    })
                    
            return result
            
    async def save_arbitrage_opportunities(self, opportunities: List[Dict[str, Any]]):
        """保存套利机会"""
        for opp in opportunities:
            # 缓存到Redis
            key = f"arbitrage:{opp['item_id']}:{datetime.now().timestamp()}"
            await self.redis.set(key, str(opp), ex=86400)  # 24小时过期
            
            # 发送通知（如果启用）
            if self.config.ENABLE_NOTIFICATIONS:
                await self._send_notification(opp)
                
    async def cleanup_old_price_history(self, days: int) -> int:
        """清理历史价格数据"""
        cutoff_date = datetime.now() - timedelta(days=days)
        
        async with self.pool.acquire() as conn:
            result = await conn.execute("""
                DELETE FROM price_histories
                WHERE recorded_at < $1
            """, cutoff_date)
            
            deleted_count = int(result.split()[-1])
            return deleted_count
            
    async def calculate_price_trends(self):
        """计算价格趋势"""
        async with self.pool.acquire() as conn:
            # 计算7日和30日平均价格
            await conn.execute("""
                UPDATE items i SET
                    avg_price_7days = (
                        SELECT AVG(price) FROM price_histories
                        WHERE item_id = i.id 
                        AND recorded_at >= NOW() - INTERVAL '7 days'
                    ),
                    avg_price_30days = (
                        SELECT AVG(price) FROM price_histories
                        WHERE item_id = i.id
                        AND recorded_at >= NOW() - INTERVAL '30 days'
                    )
            """)
            
    async def update_market_statistics(self):
        """更新市场统计数据"""
        async with self.pool.acquire() as conn:
            # 计算市场总览数据
            stats = await conn.fetchrow("""
                SELECT 
                    COUNT(*) as total_items,
                    SUM(volume_24h) as total_volume,
                    AVG(current_price) as avg_price,
                    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY current_price) as median_price
                FROM items
                WHERE current_price > 0
            """)
            
            # 缓存到Redis
            await self.redis.hset("market:stats", mapping=dict(stats))
            
    async def _send_notification(self, data: Dict[str, Any]):
        """发送通知"""
        # 这里可以实现webhook通知
        logger.info(f"发现套利机会: {data}")