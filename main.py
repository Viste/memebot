import asyncio
import logging
import sys

from dotenv import load_dotenv
from aiogram import Bot, Dispatcher
from core import setup_routers
from core.utils import config

memes = Bot(token=config.token)
load_dotenv()


async def main():
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s - %(levelname)s - %(name)s - %(message)s",
        stream=sys.stdout,
    )

    worker = Dispatcher()
    router = setup_routers()
    worker.include_router(router)
    useful_updates = worker.resolve_used_update_types()
    logging.info("Starting bot")
    await worker.start_polling(memes, allowed_updates=useful_updates)


if __name__ == '__main__':
    try:
        asyncio.run(main())
    except (KeyboardInterrupt, SystemExit):
        logging.error("Bot stopped!")
