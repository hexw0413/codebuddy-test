import os
import json
import asyncio
from datetime import datetime
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from nats.aio.client import Client as NATS


NATS_URL = os.getenv("NATS_URL", "nats://localhost:4222")


async def publish_mock_order(nc: NATS):
    payload = {
        "type": "mock_order",
        "timestamp": datetime.utcnow().isoformat(),
        "symbol": "AK-47 | Redline",
        "side": "buy",
        "price": 99.5,
        "size": 1,
    }
    await nc.publish("orders", json.dumps(payload).encode())


async def main():
    nc = NATS()
    await nc.connect(servers=[NATS_URL])

    scheduler = AsyncIOScheduler()
    scheduler.add_job(publish_mock_order, "interval", seconds=10, args=[nc])
    scheduler.start()

    try:
        while True:
            await asyncio.sleep(3600)
    finally:
        await nc.drain()


if __name__ == "__main__":
    asyncio.run(main())

