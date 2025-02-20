import base64
import json
import logging
import os
import re

import cv2
from aiogram import types
from aiogram.enums import ParseMode

from tools.ai_gpt import UserHistoryManager

logger = logging.getLogger(__name__)

class JSONObject:
    def __init__(self, dic):
        vars(self).update(dic)


def load_config():
    config_path = os.path.join(os.path.dirname(__file__), 'config.json')
    try:
        with open(config_path, 'r', encoding='utf8') as cfg_file:
            return json.load(cfg_file, object_hook=JSONObject)
    except FileNotFoundError:
        logging.error(f"Config file not found: {config_path}")
    except json.JSONDecodeError:
        logging.error(f"Error decoding JSON from the config file: {config_path}")
    return None


config = load_config()

SPAM_LINKS_REGEX = re.compile(
    r"\b(?:opensea|binance|waxu|foxu|xyz|hewuf|nft|collection|dropped|sold out|act fast|try to get)\b.*(?:https?:\/\/\S+)?",
    re.IGNORECASE)
group_id = "-1001564920057"
media_groups = {}
media_group_timers = {}


def is_spam(message: types.Message):
    return bool(message.text and SPAM_LINKS_REGEX.search(message.text))


def split_into_chunks(text: str, chunk_size: int = 4096) -> list[str]:
    return [text[i:i + chunk_size] for i in range(0, len(text), chunk_size)]


async def send_reply(message: types.Message, text: str) -> None:
    if message.chat.id != group_id:
        await message.reply("Хорошая попытка, но я сделан только для паблика @stalinfollower",
                            parse_mode=ParseMode.HTML)
    try:
        history = UserHistoryManager()
        await message.reply(text, parse_mode=ParseMode.HTML)
        await history.add_to_history(message.chat.id, "assistant", text)
    except Exception as err:
        logger.info('Exception while sending reply: %s', err)
        try:
            await message.reply(str(err), parse_mode=None)
        except Exception as error:
            logger.info('Last exception from Core: %s', error)


def extract_video_frames(video_path: str, num_frames: int = 30) -> list[str]:
    video = cv2.VideoCapture(video_path)
    base64_frames = []
    frame_count = 0

    while video.isOpened() and frame_count < num_frames:
        success, frame = video.read()
        if not success:
            break

        _, buffer = cv2.imencode(".jpg", frame)
        base64_frame = base64.b64encode(buffer.tobytes()).decode("utf-8")
        base64_frames.append(base64_frame)
        frame_count += 1

    video.release()
    return base64_frames
