#!/usr/bin/env python3
"""
CSGO2 Auto Trading - Python Data Collection Service
This service collects price data from various platforms and performs market analysis.
"""

import asyncio
import logging
import signal
import sys
from typing import Dict, Any
import schedule
import time
from datetime import datetime, timedelta

from collectors.steam_collector import SteamCollector
from collectors.buff_collector import BuffCollector
from collectors.youpin_collector import YoupinCollector
from database.db_manager import DatabaseManager
from utils.logger import setup_logger
from config import Config

class DataCollectionService:
    def __init__(self):
        self.config = Config()
        self.logger = setup_logger(__name__)
        self.db_manager = DatabaseManager(self.config.DATABASE_URL)
        
        # Initialize collectors
        self.steam_collector = SteamCollector(self.config.STEAM_API_KEY)
        self.buff_collector = BuffCollector(self.config.BUFF_API_KEY)
        self.youpin_collector = YoupinCollector(self.config.YOUPIN_API_KEY)
        
        self.running = False
        
    async def start(self):
        """Start the data collection service"""
        self.logger.info("Starting CSGO2 Data Collection Service")
        self.running = True
        
        # Setup signal handlers
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)
        
        # Initialize database
        await self.db_manager.initialize()
        
        # Schedule data collection tasks
        self._schedule_tasks()
        
        # Start the main loop
        await self._run_scheduler()
        
    def _signal_handler(self, signum, frame):
        """Handle shutdown signals"""
        self.logger.info(f"Received signal {signum}, shutting down...")
        self.running = False
        
    def _schedule_tasks(self):
        """Schedule periodic data collection tasks"""
        # Collect prices every 15 minutes
        schedule.every(15).minutes.do(self._collect_prices)
        
        # Collect market data every 30 minutes
        schedule.every(30).minutes.do(self._collect_market_data)
        
        # Analyze trends every hour
        schedule.every().hour.do(self._analyze_trends)
        
        # Clean old data daily
        schedule.every().day.at("02:00").do(self._cleanup_old_data)
        
        self.logger.info("Scheduled data collection tasks")
        
    async def _run_scheduler(self):
        """Run the scheduled tasks"""
        while self.running:
            schedule.run_pending()
            await asyncio.sleep(60)  # Check every minute
            
    def _collect_prices(self):
        """Collect price data from all platforms"""
        self.logger.info("Starting price collection")
        
        try:
            # Get items to collect prices for
            items = asyncio.run(self.db_manager.get_active_items())
            
            for item in items:
                # Collect from Steam
                steam_price = asyncio.run(self.steam_collector.get_item_price(item['market_name']))
                if steam_price:
                    asyncio.run(self.db_manager.save_price(item['id'], 'steam', steam_price))
                
                # Collect from BUFF
                buff_price = asyncio.run(self.buff_collector.get_item_price(item['market_name']))
                if buff_price:
                    asyncio.run(self.db_manager.save_price(item['id'], 'buff', buff_price))
                
                # Collect from YouPin
                youpin_price = asyncio.run(self.youpin_collector.get_item_price(item['market_name']))
                if youpin_price:
                    asyncio.run(self.db_manager.save_price(item['id'], 'youpin', youpin_price))
                
                # Rate limiting
                time.sleep(2)
                
            self.logger.info(f"Collected prices for {len(items)} items")
            
        except Exception as e:
            self.logger.error(f"Error collecting prices: {e}")
            
    def _collect_market_data(self):
        """Collect general market data"""
        self.logger.info("Collecting market data")
        
        try:
            # Collect popular items from Steam
            steam_items = asyncio.run(self.steam_collector.get_popular_items())
            
            # Save new items to database
            for item in steam_items:
                asyncio.run(self.db_manager.save_item(item))
                
            self.logger.info(f"Collected {len(steam_items)} market items")
            
        except Exception as e:
            self.logger.error(f"Error collecting market data: {e}")
            
    def _analyze_trends(self):
        """Analyze market trends"""
        self.logger.info("Analyzing market trends")
        
        try:
            # Get price data from last 24 hours
            end_time = datetime.now()
            start_time = end_time - timedelta(hours=24)
            
            items = asyncio.run(self.db_manager.get_active_items())
            
            for item in items:
                # Analyze trend for each platform
                for platform in ['steam', 'buff', 'youpin']:
                    trend = asyncio.run(
                        self.db_manager.analyze_item_trend(
                            item['id'], platform, start_time, end_time
                        )
                    )
                    
                    if trend:
                        asyncio.run(self.db_manager.save_trend(trend))
                        
            self.logger.info("Trend analysis completed")
            
        except Exception as e:
            self.logger.error(f"Error analyzing trends: {e}")
            
    def _cleanup_old_data(self):
        """Clean up old price data"""
        self.logger.info("Cleaning up old data")
        
        try:
            # Keep price data for 30 days
            cutoff_date = datetime.now() - timedelta(days=30)
            
            deleted_count = asyncio.run(self.db_manager.cleanup_old_prices(cutoff_date))
            self.logger.info(f"Cleaned up {deleted_count} old price records")
            
        except Exception as e:
            self.logger.error(f"Error cleaning up data: {e}")

async def main():
    """Main entry point"""
    service = DataCollectionService()
    await service.start()

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nShutdown requested by user")
        sys.exit(0)
    except Exception as e:
        print(f"Fatal error: {e}")
        sys.exit(1)