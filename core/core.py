import asyncio
import base64
import html
import logging
import os

import cv2
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

media_groups = {}
media_group_timers = {}


async def send_media_group(group_id, caption, message: types.Message):
    if group_id in media_groups:
        media_groups[group_id][0].caption = caption
        await memes.send_media_group(channel, media=media_groups[group_id])
        del media_groups[group_id]
        del media_group_timers[group_id]

        await message.reply("Спасибо за мем! Приходи еще")


async def group_send_delay(group_id, caption, message: types.Message):
    await asyncio.sleep(5)
    await send_media_group(group_id, caption, message)

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
        caption = f"Мем прислал: {sender_name} {sender_lastname}"

        if message.media_group_id:
            group_id = message.media_group_id

            if group_id not in media_groups:
                media_groups[group_id] = []

            media_groups[group_id].append(types.InputMediaPhoto(media=message.photo[-1].file_id))

            logging.info('id of file %s', message.photo[-1].file_id)

            if group_id not in media_group_timers:
                media_group_timers[group_id] = asyncio.create_task(group_send_delay(group_id, caption, message))

        else:
            logging.info('id of file %s', message.photo[-1].file_id)
            await memes.send_photo(channel, photo=message.photo[-1].file_id, caption=caption)
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
    group_id = message.media_group_id

    if group_id:
        if group_id not in media_groups:
            media_groups[group_id] = []

        media_groups[group_id].append(message)

        if group_id not in media_group_timers:
            media_group_timers[group_id] = asyncio.create_task(group_comment_delay(group_id))
    else:
        try:
            logging.info(
                f"Received forwarded photo from channel {message.forward_from_chat.title} in chat {message.chat.id}")

            file_info = await message.bot.get_file(message.photo[-1].file_id)
            image_url = f"https://api.telegram.org/file/bot{config.token}/{file_info.file_path}"

            comment = await openai.generate_comment_from_image(image_url, message.chat.id)

            await message.reply(comment)
        except Exception as e:
            logging.error(f"Error generating comment for single photo: {e}")
            await message.reply("Не удалось обработать фотографию. Попробуйте еще раз.")



@router.message(F.content_type.in_({'video'}), F.chat.type.in_({'group', 'supergroup'}))
async def comment_on_video(message: types.Message):
    logging.info(
        f"Received video from channel {message.forward_from_chat.title if message.forward_from_chat else 'unknown'} in chat {message.chat.id}")

    temp_folder = "temp"
    if not os.path.exists(temp_folder):
        os.makedirs(temp_folder)

    file_info = await message.bot.get_file(message.video.file_id)
    video_path = os.path.join(temp_folder, f"{message.video.file_id}.mp4")

    try:
        await message.bot.download_file(file_info.file_path, video_path)

        base64_frames = extract_video_frames(video_path)

        if base64_frames:
            comment = await openai.generate_comment_from_video_frames(base64_frames, message.chat.id)
            await message.reply(comment)
        else:
            await message.reply("Не удалось обработать видео.")

    except Exception as e:
        logging.error(f"Error during video processing: {e}")
        await message.reply("Возникла ошибка при обработке видео.")

    finally:
        if os.path.exists(video_path):
            os.remove(video_path)


def extract_video_frames(video_path: str, num_frames: int = 5) -> list[str]:
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


async def group_comment_delay(group_id):
    await asyncio.sleep(5)

    messages = media_groups.get(group_id, [])
    if not messages:
        return

    image_urls = []
    for message in messages:
        file_info = await message.bot.get_file(message.photo[-1].file_id)
        image_url = f"https://api.telegram.org/file/bot{config.token}/{file_info.file_path}"
        image_urls.append(image_url)

    comment = await openai.generate_comment_from_images(image_urls, messages[0].chat.id)

    await messages[0].reply(comment)

    del media_groups[group_id]
    del media_group_timers[group_id]

@router.message(F.reply_to_message.from_user.is_bot)
async def process_ask_chat(message: types.Message) -> None:
    logger.info("%s", message)
    text = html.escape(message.text)

    replay_text = await oai.get_resp(text, message.chat.id)
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
