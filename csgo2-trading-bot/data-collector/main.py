#!/usr/bin/env python3
"""
CSGO2 市场数据采集主程序
"""

import asyncio
import logging
import signal
import sys
from datetime import datetime
from typing import Dict, Any

from apscheduler.schedulers.asyncio import AsyncIOScheduler
from apscheduler.triggers.interval import IntervalTrigger

from collectors.steam_collector import SteamCollector
from collectors.buff_collector import BuffCollector
from collectors.youpin_collector import YouPinCollector
from database.db_manager import DatabaseManager
from config import Config
from utils.logger import setup_logger

# 设置日志
logger = setup_logger(__name__)


class DataCollectorService:
    """数据采集服务主类"""
    
    def __init__(self):
        self.config = Config()
        self.db_manager = DatabaseManager(self.config)
        self.scheduler = AsyncIOScheduler()
        
        # 初始化采集器
        self.collectors = {
            'steam': SteamCollector(self.config, self.db_manager),
            'buff': BuffCollector(self.config, self.db_manager),
            'youpin': YouPinCollector(self.config, self.db_manager),
        }
        
        self.running = False
        
    async def initialize(self):
        """初始化服务"""
        logger.info("正在初始化数据采集服务...")
        
        # 连接数据库
        await self.db_manager.connect()
        
        # 初始化各个采集器
        for name, collector in self.collectors.items():
            try:
                await collector.initialize()
                logger.info(f"{name} 采集器初始化成功")
            except Exception as e:
                logger.error(f"{name} 采集器初始化失败: {e}")
        
        # 设置定时任务
        self._setup_scheduled_jobs()
        
        logger.info("数据采集服务初始化完成")
        
    def _setup_scheduled_jobs(self):
        """设置定时任务"""
        # Steam市场数据采集 - 每5分钟
        if self.config.STEAM_ENABLED:
            self.scheduler.add_job(
                self.collect_steam_data,
                IntervalTrigger(minutes=5),
                id='steam_collector',
                name='Steam市场数据采集',
                replace_existing=True
            )
        
        # BUFF数据采集 - 每10分钟
        if self.config.BUFF_ENABLED:
            self.scheduler.add_job(
                self.collect_buff_data,
                IntervalTrigger(minutes=10),
                id='buff_collector',
                name='BUFF市场数据采集',
                replace_existing=True
            )
        
        # 悠悠有品数据采集 - 每15分钟
        if self.config.YOUPIN_ENABLED:
            self.scheduler.add_job(
                self.collect_youpin_data,
                IntervalTrigger(minutes=15),
                id='youpin_collector',
                name='悠悠有品数据采集',
                replace_existing=True
            )
        
        # 数据清理任务 - 每天凌晨3点
        self.scheduler.add_job(
            self.cleanup_old_data,
            'cron',
            hour=3,
            minute=0,
            id='data_cleanup',
            name='历史数据清理',
            replace_existing=True
        )
        
        # 数据分析任务 - 每小时
        self.scheduler.add_job(
            self.analyze_market_data,
            IntervalTrigger(hours=1),
            id='market_analysis',
            name='市场数据分析',
            replace_existing=True
        )
        
    async def collect_steam_data(self):
        """采集Steam市场数据"""
        try:
            logger.info("开始采集Steam市场数据...")
            items = await self.collectors['steam'].collect_market_data()
            logger.info(f"Steam市场数据采集完成，共采集 {len(items)} 个物品")
        except Exception as e:
            logger.error(f"Steam数据采集失败: {e}")
            
    async def collect_buff_data(self):
        """采集BUFF市场数据"""
        try:
            logger.info("开始采集BUFF市场数据...")
            items = await self.collectors['buff'].collect_market_data()
            logger.info(f"BUFF市场数据采集完成，共采集 {len(items)} 个物品")
        except Exception as e:
            logger.error(f"BUFF数据采集失败: {e}")
            
    async def collect_youpin_data(self):
        """采集悠悠有品市场数据"""
        try:
            logger.info("开始采集悠悠有品市场数据...")
            items = await self.collectors['youpin'].collect_market_data()
            logger.info(f"悠悠有品数据采集完成，共采集 {len(items)} 个物品")
        except Exception as e:
            logger.error(f"悠悠有品数据采集失败: {e}")
            
    async def cleanup_old_data(self):
        """清理历史数据"""
        try:
            logger.info("开始清理历史数据...")
            deleted_count = await self.db_manager.cleanup_old_price_history(days=90)
            logger.info(f"历史数据清理完成，删除 {deleted_count} 条记录")
        except Exception as e:
            logger.error(f"数据清理失败: {e}")
            
    async def analyze_market_data(self):
        """分析市场数据"""
        try:
            logger.info("开始分析市场数据...")
            
            # 计算价格趋势
            await self.db_manager.calculate_price_trends()
            
            # 识别套利机会
            opportunities = await self.identify_arbitrage_opportunities()
            if opportunities:
                logger.info(f"发现 {len(opportunities)} 个套利机会")
                await self.db_manager.save_arbitrage_opportunities(opportunities)
            
            # 更新市场统计
            await self.db_manager.update_market_statistics()
            
            logger.info("市场数据分析完成")
        except Exception as e:
            logger.error(f"市场分析失败: {e}")
            
    async def identify_arbitrage_opportunities(self) -> list:
        """识别套利机会"""
        opportunities = []
        
        try:
            # 获取所有物品的跨平台价格
            items = await self.db_manager.get_cross_platform_prices()
            
            for item in items:
                prices = item.get('prices', {})
                if len(prices) < 2:
                    continue
                    
                # 找出最低和最高价格
                min_platform = min(prices, key=prices.get)
                max_platform = max(prices, key=prices.get)
                
                min_price = prices[min_platform]
                max_price = prices[max_platform]
                
                # 计算利润率
                profit_rate = (max_price - min_price) / min_price * 100
                
                # 如果利润率超过阈值，记录套利机会
                if profit_rate > self.config.MIN_ARBITRAGE_PROFIT:
                    opportunities.append({
                        'item_id': item['id'],
                        'item_name': item['name'],
                        'buy_platform': min_platform,
                        'buy_price': min_price,
                        'sell_platform': max_platform,
                        'sell_price': max_price,
                        'profit_rate': profit_rate,
                        'timestamp': datetime.now()
                    })
                    
        except Exception as e:
            logger.error(f"识别套利机会失败: {e}")
            
        return opportunities
        
    async def start(self):
        """启动服务"""
        self.running = True
        await self.initialize()
        
        # 启动调度器
        self.scheduler.start()
        logger.info("定时任务调度器已启动")
        
        # 立即执行一次数据采集
        await self.initial_collection()
        
        # 保持运行
        try:
            while self.running:
                await asyncio.sleep(1)
        except KeyboardInterrupt:
            logger.info("收到中断信号，正在停止服务...")
            await self.stop()
            
    async def initial_collection(self):
        """初始数据采集"""
        logger.info("开始初始数据采集...")
        
        tasks = []
        if self.config.STEAM_ENABLED:
            tasks.append(self.collect_steam_data())
        if self.config.BUFF_ENABLED:
            tasks.append(self.collect_buff_data())
        if self.config.YOUPIN_ENABLED:
            tasks.append(self.collect_youpin_data())
            
        if tasks:
            await asyncio.gather(*tasks, return_exceptions=True)
            
        logger.info("初始数据采集完成")
        
    async def stop(self):
        """停止服务"""
        self.running = False
        
        # 停止调度器
        self.scheduler.shutdown()
        
        # 关闭采集器
        for collector in self.collectors.values():
            await collector.close()
            
        # 关闭数据库连接
        await self.db_manager.close()
        
        logger.info("数据采集服务已停止")


def signal_handler(signum, frame):
    """信号处理器"""
    logger.info(f"收到信号 {signum}")
    sys.exit(0)


async def main():
    """主函数"""
    # 设置信号处理
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    # 创建并启动服务
    service = DataCollectorService()
    await service.start()


if __name__ == "__main__":
    asyncio.run(main())