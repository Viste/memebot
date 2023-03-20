import logging

from aiogram import types, F, Router
from aiogram.filters.command import Command
from main import memes
from core.utils import config

logger = logging.getLogger("__name__")
router = Router()
channel = config.channel


@router.message(Command(commands="start", ignore_case=True), F.chat.type == "private")
async def start(message: types.Message):
    first_name = message.chat.first_name

    await message.reply(f"Привет {first_name}, тут ты можешь отправить нам мемес. Принимаю только видосики и картинощки")


@router.message(F.content_type.in_({'photo'}), F.chat.type == "private")
async def work_send_tax(message: types.Message):
    logging.info('info about message %s', message)
    logging.info('id of file %s', message.photo[-1].file_id)
    if message.chat.first_name is None:
        sender_name = message.chat.username
    elif message.chat.first_name == "":
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
