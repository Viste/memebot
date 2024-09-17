import logging
import jwt
import os

from aiogram import types, F, Router
from aiogram.filters.command import Command
from main import memes
from core.utils import config
from core.utils import GROUP_ID
from core.utils import is_spam
import requests

logger = logging.getLogger("__name__")
router = Router()
channel = config.channel


@router.message(Command(commands="start", ignore_case=True), F.chat.type == "private")
async def start_handler(message: types.Message):
    first_name = message.from_user.first_name
    last_name = message.from_user.last_name
    username = message.from_user.username
    telegram_id = str(message.from_user.id)

    await message.reply(f"Привет {first_name}, тут ты можешь отправить нам мемес. Принимаю только видосики и картинощки\n"
                            f"А еще связать свой аккаунт с сайтом dev-vlab.ru через команду /start auth")


@router.message(F.content_type.in_({'photo'}), F.chat.type == "private")
async def work_send_tax(message: types.Message):
    uid = message.from_user.id
    if uid in config.banned_user_ids:
        text = "не хочу с тобой разговаривать"
        await message.reply(text, parse_mode=None)
    else:
        logging.info('info about message %s', message)
        logging.info('id of file %s', message.photo[-1].file_id)
        if message.chat.first_name is None:
            sender_name = message.chat.username
        elif message.chat.first_name == "":
            sender_name = message.chat.username
        elif message.chat.first_name == "\xad":
            sender_name = message.chat.username
        else:
            sender_name = message.chat.first_name

        if message.chat.last_name is None:
            sender_lastname = ' '
        else:
            sender_lastname = message.chat.last_name

        text = f"Мем прислал: {sender_name} {sender_lastname}"
        await memes.send_photo(channel, photo=message.photo[-1].file_id, caption=text)
        await message.reply("Спасибо за мем! Пока-пока")


@router.message(F.content_type.in_({'video'}), F.chat.type == "private")
async def work_send_demo(message: types.Message):
    uid = message.from_user.id
    if uid in config.banned_user_ids:
        text = "не хочу с тобой разговаривать"
        await message.reply(text, parse_mode=None)
    else:
        logging.info('info about message %s', message)
        logging.info('id of file %s', message.video.file_id)
        sender_name = message.chat.first_name
        if message.chat.last_name is None:
            sender_lastname = ' '
        else:
            sender_lastname = message.chat.last_name
        content = message.video.file_id
        text = f"Мем прислал: {sender_name} {sender_lastname}"
        await memes.send_video(channel, video=content, caption=text)
        await message.reply("Спасибо за мем! Пока-пока")


@router.message(F.chat.id == GROUP_ID)
async def handle_group_messages(message: types.Message):
    if is_spam(message):
        await message.delete()
        await memes.ban_chat_member(chat_id=GROUP_ID, user_id=message.from_user.id)
        logging.info(f"User {message.from_user.id} banned for spamming")
        await message.answer(f"Пользователь {message.from_user.first_name} был заблокирован за спам.")
    else:
        logging.info(f"Received message from {message.from_user.first_name}: {message.text}")