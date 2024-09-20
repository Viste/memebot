import html
import logging

from aiogram import types, F, Router
from aiogram.filters.command import Command

from main import memes
from tools.mentality import OpenAI
from tools.utils import OpenAIVision
from tools.utils import config, split_into_chunks
from tools.utils import is_spam, group_id, send_reply

logger = logging.getLogger(__name__)
router = Router()
channel = config.channel
openai = OpenAIVision()
oai = OpenAI()

@router.message(Command(commands="start", ignore_case=True), F.chat.type == "private")
async def start_handler(message: types.Message):
    first_name = message.from_user.first_name

    await message.reply(f"Привет {first_name}, тут ты можешь отправить нам мемес. Принимаю только видосики и картинощки")


@router.message(F.content_type.in_({'photo'}), F.chat.type == "private")
async def work_send_meme(message: types.Message):
    uid = message.from_user.id
    if uid in config.banned_user_ids:
        text = "не хочу с тобой разговаривать"
        await message.reply(text, parse_mode=None)
    else:
        logging.info('info about message %s', message)

        if message.chat.first_name is None or message.chat.first_name == "" or message.chat.first_name == "\xad":
            sender_name = message.chat.username
        else:
            sender_name = message.chat.first_name

        sender_lastname = message.chat.last_name if message.chat.last_name else ' '

        text = f"Мем прислал: {sender_name} {sender_lastname}"

        if message.media_group_id:
            media_group = []
            for photo in message.photo:
                media_group.append(types.InputMediaPhoto(media=photo.file_id, caption=text))
                logging.info('id of file %s', photo.file_id)

            await memes.send_media_group(channel, media=media_group)
            await message.reply("Спасибо за мемы! Пока-пока")
        else:
            logging.info('id of file %s', message.photo[-1].file_id)
            await memes.send_photo(channel, photo=message.photo[-1].file_id, caption=text)
            await message.reply("Спасибо за мем! Пока-пока")


@router.message(F.content_type.in_({'video'}), F.chat.type == "private")
async def work_send_meme_video(message: types.Message):
    uid = message.from_user.id
    if uid in config.banned_user_ids:
        text = "не хочу с тобой разговаривать"
        await message.reply(text, parse_mode=None)
    else:
        logging.info('info about message %s', message)
        logging.info('id of file %s', message.video.file_id)
        if message.chat.first_name is None:
            sender_name = message.chat.username
        else:
            sender_name = message.chat.first_name
        if message.chat.last_name is None:
            sender_lastname = ' '
        else:
            sender_lastname = message.chat.last_name
        content = message.video.file_id
        text = f"Мем прислал: {sender_name} {sender_lastname}"
        await memes.send_video(channel, video=content, caption=text)
        await message.reply("Спасибо за мем! Пока-пока")


@router.message(F.content_type.in_({'photo'}), F.chat.type.in_({'group', 'supergroup'}))
async def comment_on_photo(message: types.Message):
    logging.info(f"Received forwarded photo from channel {message.forward_from_chat.title} in chat {message.chat.id}")

    file_info = await message.bot.get_file(message.photo[-1].file_id)
    image_url = f"https://api.telegram.org/file/bot{config.token}/{file_info.file_path}"

    comment = await openai.generate_comment_from_image(image_url)

    await message.reply(comment)


@router.message(F.reply_to_message.from_user.is_bot)
async def process_ask_chat(message: types.Message) -> None:
    logger.info("%s", message)
    text = html.escape(message.text)

    replay_text = await oai.get_resp(text, message.from_user.id)
    chunks = split_into_chunks(replay_text)
    for index, chunk in enumerate(chunks):
        if index == 0:
            await send_reply(message, chunk)


@router.message(F.chat.type.in_({'group', 'supergroup'}))
async def handle_group_messages(message: types.Message):
    if is_spam(message):
        await message.delete()
        await memes.ban_chat_member(chat_id=group_id, user_id=message.from_user.id)
        logging.info(f"User {message.from_user.id} banned for spamming")
        await message.answer(f"Пользователь {message.from_user.first_name} был заблокирован за спам.")
    else:
        logging.info(f"Received message from {message.from_user.first_name}: {message.text}")
