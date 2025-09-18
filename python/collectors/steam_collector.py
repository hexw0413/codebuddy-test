from typing import Dict, List, Optional, Any
import urllib.parse
from .base_collector import BaseCollector

class SteamCollector(BaseCollector):
    """Steam Community Market data collector"""
    
    def __init__(self, api_key: Optional[str] = None):
        super().__init__(api_key)
        self.base_url = "https://steamcommunity.com"
        self.api_base_url = "https://api.steampowered.com"
        self.rate_limit_delay = 1.5  # Steam is more lenient
        
    def get_platform_name(self) -> str:
        return "steam"
        
    async def get_item_price(self, market_name: str) -> Optional[Dict[str, Any]]:
        """Get current price for an item from Steam Market"""
        encoded_name = urllib.parse.quote(market_name)
        url = f"{self.base_url}/market/priceoverview/"
        
        params = {
            'appid': 730,  # CS:GO
            'currency': 1,  # USD
            'market_hash_name': encoded_name
        }
        
        try:
            data = await self._make_request(url, params=params)
            
            if not data or not data.get('success'):
                return None
                
            # Parse price (remove $ symbol and convert to float)
            lowest_price_str = data.get('lowest_price', '$0.00')
            median_price_str = data.get('median_price', '$0.00')
            volume_str = data.get('volume', '0')
            
            try:
                lowest_price = float(lowest_price_str.replace('$', '').replace(',', ''))
                median_price = float(median_price_str.replace('$', '').replace(',', '')) if median_price_str != '$0.00' else lowest_price
                volume = int(volume_str.replace(',', '')) if volume_str.replace(',', '').isdigit() else 0
            except (ValueError, AttributeError):
                return None
                
            return self._normalize_price_data({
                'price': lowest_price,
                'median_price': median_price,
                'volume': volume,
                'currency': 'USD'
            })
            
        except Exception as e:
            self.logger.error(f"Error getting Steam price for {market_name}: {e}")
            return None
            
    async def get_popular_items(self, limit: int = 100) -> List[Dict[str, Any]]:
        """Get popular items from Steam Market"""
        url = f"{self.base_url}/market/search/render/"
        
        params = {
            'appid': 730,
            'norender': 1,
            'count': min(limit, 100),  # Steam limits to 100 per request
            'sort_column': 'quantity',
            'sort_dir': 'desc'
        }
        
        try:
            data = await self._make_request(url, params=params)
            
            if not data or not data.get('success'):
                return []
                
            items = []
            results = data.get('results', [])
            
            for item_data in results:
                # Extract item information
                item = {
                    'name': item_data.get('name', ''),
                    'market_name': item_data.get('hash_name', ''),
                    'icon_url': f"https://community.cloudflare.steamstatic.com/economy/image/{item_data.get('icon_url', '')}",
                    'type': self._extract_item_type(item_data.get('type', '')),
                    'weapon': self._extract_weapon_name(item_data.get('name', '')),
                    'exterior': self._extract_exterior(item_data.get('name', '')),
                    'rarity': self._extract_rarity(item_data.get('name', '')),
                    'collection': '',  # Not available in search results
                }
                
                normalized_item = self._normalize_item_data(item)
                if normalized_item:
                    items.append(normalized_item)
                    
            return items
            
        except Exception as e:
            self.logger.error(f"Error getting popular Steam items: {e}")
            return []
            
    async def get_item_history(self, market_name: str, days: int = 7) -> List[Dict[str, Any]]:
        """Get price history for an item (Steam doesn't provide public API for this)"""
        # Steam's price history is not publicly available via API
        # Would need to scrape the market page or use unofficial APIs
        self.logger.info(f"Price history not available for Steam items via public API")
        return []
        
    async def get_user_inventory(self, steam_id: str) -> List[Dict[str, Any]]:
        """Get user's CS:GO inventory"""
        url = f"{self.base_url}/inventory/{steam_id}/730/2"
        
        params = {
            'l': 'english',
            'count': 5000
        }
        
        try:
            data = await self._make_request(url, params=params)
            
            if not data or not data.get('success'):
                return []
                
            assets = data.get('assets', [])
            descriptions = data.get('descriptions', [])
            
            # Create a mapping of classid+instanceid to description
            desc_map = {}
            for desc in descriptions:
                key = f"{desc.get('classid')}_{desc.get('instanceid')}"
                desc_map[key] = desc
                
            inventory_items = []
            for asset in assets:
                key = f"{asset.get('classid')}_{asset.get('instanceid')}"
                desc = desc_map.get(key, {})
                
                item = {
                    'asset_id': asset.get('assetid'),
                    'name': desc.get('name', ''),
                    'market_name': desc.get('market_hash_name', ''),
                    'icon_url': f"https://community.cloudflare.steamstatic.com/economy/image/{desc.get('icon_url', '')}",
                    'type': desc.get('type', ''),
                    'tradable': desc.get('tradable', 0) == 1,
                    'marketable': desc.get('marketable', 0) == 1,
                    'amount': int(asset.get('amount', 1))
                }
                
                inventory_items.append(item)
                
            return inventory_items
            
        except Exception as e:
            self.logger.error(f"Error getting Steam inventory for {steam_id}: {e}")
            return []
            
    def _extract_item_type(self, type_str: str) -> str:
        """Extract item type from description"""
        type_mapping = {
            'rifle': 'Rifle',
            'pistol': 'Pistol',
            'sniper': 'Sniper Rifle',
            'shotgun': 'Shotgun',
            'smg': 'SMG',
            'machinegun': 'Machine Gun',
            'knife': 'Knife',
            'gloves': 'Gloves',
            'sticker': 'Sticker',
            'music': 'Music Kit'
        }
        
        type_lower = type_str.lower()
        for key, value in type_mapping.items():
            if key in type_lower:
                return value
                
        return 'Unknown'
        
    def _extract_weapon_name(self, name: str) -> str:
        """Extract weapon name from item name"""
        # Common weapon names
        weapons = ['AK-47', 'M4A4', 'M4A1-S', 'AWP', 'Glock-18', 'USP-S', 'Desert Eagle', 
                  'Karambit', 'Bayonet', 'Butterfly Knife', 'Flip Knife']
        
        for weapon in weapons:
            if weapon in name:
                return weapon
                
        # Try to extract first word as weapon name
        parts = name.split(' ')
        if len(parts) > 0:
            return parts[0]
            
        return 'Unknown'
        
    def _extract_exterior(self, name: str) -> str:
        """Extract exterior condition from item name"""
        exteriors = ['Factory New', 'Minimal Wear', 'Field-Tested', 'Well-Worn', 'Battle-Scarred']
        
        for exterior in exteriors:
            if exterior in name:
                return exterior
                
        return 'Unknown'
        
    def _extract_rarity(self, name: str) -> str:
        """Extract rarity from item name (limited information available)"""
        # This would typically come from game data or additional API calls
        return 'Unknown'