import asyncio
import logging
import sys

from aiogram import Bot, Dispatcher
from dotenv import load_dotenv

from core import setup_routers
from core.web import start_web_app
from tools.ai_gpt import OpenAI, MemeCommentHistoryManager, CommentMemeManager, UserHistoryManager
from tools.utils import config

memes = Bot(token=config.token)
load_dotenv()
user_history_manager = UserHistoryManager()
meme_history_manager = MemeCommentHistoryManager()
comment_to_meme_manager = CommentMemeManager()
openai = OpenAI()


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
    logging.info("Starting healthz status")
    await start_web_app()
    logging.info("Starting bot")
    await worker.start_polling(memes, allowed_updates=useful_updates)


if __name__ == '__main__':
    try:
        asyncio.run(main())
    except (KeyboardInterrupt, SystemExit):
        logging.error("Bot stopped!")
