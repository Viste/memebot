import logging
import jwt
import os

from aiogram import types, F, Router
from aiogram.filters.command import Command
from main import memes
from core.utils import config
import requests

logger = logging.getLogger("__name__")
router = Router()
channel = config.channel
API_URL = "https://dev-vlab.ru/api/telegram_user"
TELEGRAM_BOT_SECRET = os.getenv('TELEGRAM_BOT_SECRET')


def generate_jwt():
    payload = {
        "iss": "telegram_bot"
    }
    token = jwt.encode(payload, TELEGRAM_BOT_SECRET, algorithm="HS256")
    return token


@router.message(Command(commands="start", ignore_case=True), F.chat.type == "private")
async def start_handler(message: types.Message):
    first_name = message.from_user.first_name
    last_name = message.from_user.last_name
    username = message.from_user.username
    telegram_id = str(message.from_user.id)

    args = message.text.split(maxsplit=1)
    if len(args) > 1:
        payload = args[1]
        if payload == 'auth':
            telegram_user_data = {'telegram_id': telegram_id, 'username': username, 'first_name': first_name, 'last_name': last_name}
            headers = {'Authorization': f'Bearer {generate_jwt()}'}
            response = requests.post(API_URL, json=telegram_user_data, headers=headers)

            if response.status_code == 200:
                await message.answer("Вы успешно авторизовались через Telegram!")
            else:
                await message.answer("Произошла ошибка при регистрации. Попробуйте позже.")
    else:
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
