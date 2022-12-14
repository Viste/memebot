import logging
from utils import *
from aiogram import Bot, Dispatcher, types
from aiogram.contrib.fsm_storage.memory import MemoryStorage
from aiogram.utils import executor

logging.basicConfig(level=logging.INFO)
bot = Bot(token=cfg.token)
storage = MemoryStorage()
worker = Dispatcher(bot, storage=storage)
channel = cfg.channel

@worker.message_handler(commands='start', chat_type='private')
async def start(message: types.Message):
    await message.reply(f"Привет {message['from'].first_name}, тут ты можешь отправить нам мемес. Принимаю только видосики и картинощки")

@worker.message_handler(content_types="photo", chat_type='private')
async def work_send_tax(message: types.Message):
    logging.info('info about message %s', message)
    logging.info('id of file %s', message.photo[-1].file_id)
    sender_name = message.chat.first_name
    text = f"Мем прислал: {sender_name}"
    await bot.send_photo(channel, photo=message.photo[-1].file_id, caption=text)
    await message.reply("Спасибо за мем! Пока-пока")

@worker.message_handler(chat_type='private', content_types="video")
async def work_send_demo(message: types.Message):
    logging.info('info about message %s', message)
    logging.info('id of file %s', message.video.file_id)
    sender_name = message.chat.first_name
    content = message.video.file_id
    text = f"Мем прислал: {sender_name}."
    await bot.send_video(channel, video=content, caption=text)
    await message.reply("Спасибо за мем! Пока-пока")

if __name__ == '__main__':
    executor.start_polling(worker, skip_updates=True)