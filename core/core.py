import logging

from aiogram import types, F, Router
from aiogram.filters.command import Command
from main import memes
from core.utils import config
import requests

logger = logging.getLogger("__name__")
router = Router()
channel = config.channel


@router.message(Command(commands=['link']), F.chat.type == "private")
async def link_account(message: types.Message):
    user = message.from_user
    profile_photos = await memes.get_user_profile_photos(user.id)
    if profile_photos.total_count > 0:
        photo_file_id = profile_photos.photos[0][-1].file_id
        file = await memes.get_file(photo_file_id)
        file_url = f"https://api.telegram.org/file/bot{memes.token}/{file.file_path}"

        response = requests.post('https://dev-vlab.ru/update_telegram_profile', json={
            'telegram_id': user.id,
            'username': user.username,
            'first_name': user.first_name,
            'last_name': user.last_name,
            'profile_picture': file_url
        })

        if response.status_code == 200:
            await message.reply("Account linked successfully!")
        else:
            await message.reply("Failed to link your account.")
    else:
        await message.reply("You don't have a profile picture.")


@router.message(Command(commands="start", ignore_case=True), F.chat.type == "private")
async def start(message: types.Message):
    first_name = message.chat.first_name

    await message.reply(f"Привет {first_name}, тут ты можешь отправить нам мемес. Принимаю только видосики и картинощки\n"
                        f"А еще связать свой аккаунт с сайтом dev-vlab.ru через команду /link")


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
