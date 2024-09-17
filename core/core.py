import logging

from aiogram import types, F, Router
from aiogram.filters.command import Command

from main import memes
from tools.utils import config
from tools.utils import generate_comment_from_image
from tools.utils import is_spam

logger = logging.getLogger("__name__")
router = Router()
channel = config.channel


@router.message(Command(commands="start", ignore_case=True), F.chat.type == "private")
async def start_handler(message: types.Message):
    first_name = message.from_user.first_name

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


@router.message(F.chat.id == -1001564920057, F.chat.type.in_({'group', 'supergroup'}))
async def handle_group_messages(message: types.Message):
    if is_spam(message):
        await message.delete()
        await memes.ban_chat_member(chat_id=-1001564920057, user_id=message.from_user.id)
        logging.info(f"User {message.from_user.id} banned for spamming")
        await message.answer(f"Пользователь {message.from_user.first_name} был заблокирован за спам.")
    else:
        logging.info(f"Received message from {message.from_user.first_name}: {message.text}")

# @router.message(F.chat.type.in_({'group', 'supergroup'}))
# async def log_all_group_messages(message: types.Message):
#     logging.info(f"Received a message: {message}")

@router.message(F.content_type.in_({'photo'}), F.chat.id == -1001564920057, F.chat.type.in_({'group', 'supergroup'}))
async def comment_on_photo(message: types.Message):
    logging.info('Received a photo in chat %s from user %s', message.chat.id, message.from_user.id)

    file_info = await message.bot.get_file(message.photo[-1].file_id)
    image_url = f"https://api.telegram.org/file/bot{config.token}/{file_info.file_path}"

    comment = await generate_comment_from_image(image_url)

    await message.reply(f"Комментарий на фото: {comment}")
