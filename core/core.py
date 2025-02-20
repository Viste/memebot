import asyncio
import html
import logging

from aiogram import types, F, Router
from aiogram.filters.command import Command

from main import memes
from tools.ai_gpt import OpenAI
from tools.utils import config
from tools.utils import is_spam, group_id
from tools.utils import media_groups, media_group_timers
from tools.utils import send_reply, split_into_chunks

logger = logging.getLogger(__name__)
router = Router()
channel = config.channel
openai = OpenAI()


async def group_comment_delay(group_id_local):
    await asyncio.sleep(5)

    messages = media_groups.get(group_id_local, [])
    if not messages:
        logger.warning(f"No messages found for group ID {group_id_local}")
        return

    image_urls = []
    for message in messages:
        try:
            file_info = await message.bot.get_file(message.photo[-1].file_id)
            image_url = f"https://api.telegram.org/file/bot{config.token}/{file_info.file_path}"
            image_urls.append(image_url)
        except Exception as e:
            logger.error(f"Error getting file info for message: {e}")

    if image_urls:
        comment = await openai.generate_comment_from_images(image_urls, messages[0].chat.id)
        await messages[0].reply(comment)
    else:
        logger.warning(f"No valid image URLs found for group ID {group_id_local}")

    del media_groups[group_id_local]
    del media_group_timers[group_id_local]


async def send_media_group(group_id_local, caption, message: types.Message):
    if group_id_local in media_groups:
        media_groups[group_id_local][0].caption = caption
        try:
            await memes.send_media_group(channel, media=media_groups[group_id_local])
            del media_groups[group_id_local]
            del media_group_timers[group_id_local]
            await message.reply("Спасибо за мем! Приходи еще")
        except Exception as e:
            logger.error(f"Error sending media group: {e}")


async def group_send_delay(group_id_delay, caption, message: types.Message):
    await asyncio.sleep(5)
    await send_media_group(group_id_delay, caption, message)


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

        sender_name = message.chat.username if not message.chat.first_name or message.chat.first_name == "\xad" else message.chat.first_name
        sender_lastname = message.chat.last_name if message.chat.last_name else ' '
        caption = f"Мем прислал: {sender_name} {sender_lastname}"

        if message.media_group_id:
            group_id_local = message.media_group_id

            if group_id_local not in media_groups:
                media_groups[group_id_local] = []

            media_groups[group_id_local].append(types.InputMediaPhoto(media=message.photo[-1].file_id))

            logging.info('id of file %s', message.photo[-1].file_id)

            if group_id_local not in media_group_timers:
                media_group_timers[group_id_local] = asyncio.create_task(
                    group_send_delay(group_id_local, caption, message))

        else:
            logging.info('id of file %s', message.photo[-1].file_id)
            try:
                await memes.send_photo(channel, photo=message.photo[-1].file_id, caption=caption)
                await message.reply("Спасибо за мем! Пока-пока")
            except Exception as e:
                logger.error(f"Error sending photo: {e}")


@router.message(F.content_type.in_({'video'}), F.chat.type == "private")
async def work_send_meme_video(message: types.Message):
    uid = message.from_user.id
    if uid in config.banned_user_ids:
        text = "не хочу с тобой разговаривать"
        await message.reply(text, parse_mode=None)
    else:
        logging.info('info about message %s', message)
        logging.info('id of file %s', message.video.file_id)
        sender_name = message.chat.username if not message.chat.first_name else message.chat.first_name
        sender_lastname = ' ' if not message.chat.last_name else message.chat.last_name
        content = message.video.file_id
        text = f"Мем прислал: {sender_name} {sender_lastname}"
        try:
            await memes.send_video(channel, video=content, caption=text)
            await message.reply("Спасибо за мем! Пока-пока")
        except Exception as e:
            logger.error(f"Error sending video: {e}")


@router.message(F.content_type.in_({'photo'}), F.chat.type.in_({'group', 'supergroup'}))
async def comment_on_photo(message: types.Message):
    msg_group_id = message.media_group_id
    logging.info('info about message %s', message)
    if not message.forward_from_chat or message.forward_from_chat.id != channel:
        await message.reply("Хорошая попытка, но я сделан только для паблика @stalinfollower")
        return
    elif msg_group_id:
        if msg_group_id not in media_groups:
            media_groups[msg_group_id] = []

        media_groups[msg_group_id].append(message)

        if msg_group_id not in media_group_timers:
            media_group_timers[msg_group_id] = asyncio.create_task(group_comment_delay(msg_group_id))
    else:
        try:
            logging.info(
                f"Received forwarded photo from channel {message.forward_from_chat.title if message.forward_from_chat else 'unknown'} and user {message.from_user.username} in chat {message.chat.id}")

            file_info = await message.bot.get_file(message.photo[-1].file_id)
            image_url = f"https://api.telegram.org/file/bot{config.token}/{file_info.file_path}"
            logger.info(f"Image URL: {image_url}")

            try:
                comment = await openai.generate_comment_from_image(image_url, message.chat.id)
                await message.reply(comment)
                logger.info(f"Received comment from OpenAI: {comment}")
            except Exception as e:
                logger.error(f"Error generating comment for single photo: {e}")
                await message.reply("Не удалось обработать фотографию. Попробуйте еще раз.")
        except Exception as e:
            logger.error(f"Error getting file info for single photo: {e}")
            await message.reply("Не удалось получить информацию о фотографии. Попробуйте еще раз.")


# @router.message(F.content_type.in_({'video'}), F.chat.type.in_({'group', 'supergroup'}))
# async def comment_on_video(message: types.Message):
#    logging.info(
#        f"Received video from channel {message.forward_from_chat.title if message.forward_from_chat else 'unknown'} in chat {message.chat.id}")

#    temp_folder = "temp"
#    if not os.path.exists(temp_folder):
#        os.makedirs(temp_folder)

#    file_info = await message.bot.get_file(message.video.file_id)
#    video_path = os.path.join(temp_folder, f"{message.video.file_id}.mp4")

#    try:
#        await message.bot.download_file(file_info.file_path, video_path)

#        base64_frames = extract_video_frames(video_path)

#        if base64_frames:
#            comment = await openai.generate_comment_from_video_frames(base64_frames, message.chat.id)
#            await message.reply(comment)
#        else:
#            await message.reply("Не удалось обработать видео.")

#    except Exception as e:
#        logging.error(f"Error during video processing: {e}")
#        await message.reply("Возникла ошибка при обработке видео.")

#    finally:
#        if os.path.exists(video_path):
#            os.remove(video_path)


@router.message(F.reply_to_message.from_user.is_bot)
async def process_ask_chat(message: types.Message) -> None:
    logger.info("%s", message)
    text = html.escape(message.text)
    if message.chat.id != "-1001564920057":
        await message.reply("Хорошая попытка, но я сделан только для паблика @stalinfollower")
        return

    try:
        replay_text = await openai.get_resp(text, message.chat.id)
        chunks = split_into_chunks(replay_text)
        for index, chunk in enumerate(chunks):
            if index == 0:
                await send_reply(message, chunk)
    except Exception as e:
        logger.error(f"Error processing chat response: {e}")
        await message.reply("Не удалось обработать ваш запрос. Попробуйте позже.")


@router.message(F.chat.type.in_({'group', 'supergroup'}))
async def handle_group_messages(message: types.Message):
    logger.info("%s", message)
    logging.info(
        f"Received message in chatid {message.chat.id}, chat name: {message.chat.title} from {message.from_user.first_name} {message.from_user.username}: {message.text}")

    if message.photo or message.video or message.forward_from_chat:
        return

    if message.chat.id != "-1001564920057":
        await message.reply("Хорошая попытка, но я сделана только для паблика @stalinfollower")
        return

    if is_spam(message):
        await message.delete()
        try:
            await memes.ban_chat_member(chat_id=group_id, user_id=message.from_user.id)
            logging.info(f"User {message.from_user.id} banned for spamming")
            await message.answer(f"Пользователь {message.from_user.first_name} был заблокирован за спам.")
        except Exception as e:
            logger.error(f"Error banning user {message.from_user.id}: {e}")
