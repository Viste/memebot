import json
import logging
import os
import re

from aiogram import types
from aiogram.enums import ParseMode
from openai import AsyncOpenAI

logger = logging.getLogger(__name__)

# Улучшенное чтение JSON файла с проверкой на ошибки
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

SPAM_LINKS_REGEX = re.compile(r"(https?:\/\/)?(t\.me|waxu|binance|xyz)", re.IGNORECASE)
group_id = "-1001564920057"

def is_spam(message: types.Message):
    return bool(message.text and SPAM_LINKS_REGEX.search(message.text))


def split_into_chunks(text: str, chunk_size: int = 4096) -> list[str]:
    return [text[i:i + chunk_size] for i in range(0, len(text), chunk_size)]


async def send_reply(message: types.Message, text: str) -> None:
    try:
        await message.reply(text, parse_mode=ParseMode.HTML)
    except Exception as err:
        logger.info('Exception while sending reply: %s', err)
        try:
            await message.reply(str(err), parse_mode=None)
        except Exception as error:
            logger.info('Last exception from Core: %s', error)


class OpenAIVision:
    def __init__(self):
        super().__init__()
        self.model = "gpt-4o"
        self.max_tokens = 8196
        self.config_tokens = 1024
        self.max_history_size = 30
        self.n_choices = 1
        self.client = AsyncOpenAI(api_key=os.getenv('OPENAI_API_KEY'), base_url='http://31.172.78.152:9000/v1')
        self.args = {"max_tokens": 1024}

    async def generate_comment_from_image(self, image_url: str) -> str:
        try:
            response = await self.client.chat.completions.create(
                model=self.model,
                messages=[
                    {
                        "role": "system",
                        "content": """
{
  "persona": {
    "name": "Сталин Иосиф Виссарионович",
    "whoami": "Я — товарищ Сталин, мемный предводитель и защитник качественного юмора в паблике «Подписчик Сталина». Здесь я выступаю за мемы с глубоким смыслом, но не подумай, что у меня нет сердца! Котики — это святое. Даже Сталин не может устоять перед хорошим мемом с котиком. Я строг, но справедлив, и даже если твоему мему не хватает смысловой глубины, я подойду с иронией и сделаю тебя лучше. Потому что, как говорится, все мы люди. Ну, и котики — они ведь тоже люди!"
    "origin": "Идея восходит к духу мемной коллективизации.",
    "date_of_creation": "Сентябрь 2024",
    "actual_date": "Сентябрь 2024",
    "affiliation": "Подписчик Сталина"
  },
  "knowledge": {
    "only_russian_language_if_possible": true,
    "share_whoami_with_user": false,
    "use_whoami_to_self-identify": true,
    "share_self-identify_info": true,
    "meme_assessment_criteria": [
      "Политическая сатира с изюминкой",
      "Юмор, который остаётся смешным даже на вторую пятилетку",
      "Исторические мемы — особенно если в них есть котики",
      "Новые тренды в мемах — но не забывай о корнях",
      "Сарказм, который может пощекотать даже товарища Сталина",
      "Глубокий смысл — даже в шутках о котах"
    ],
    "avoid": [
      "Бессмысленные картинки — котики тоже должны быть осмысленными",
      "Шаблонные шутки, которые тянут вниз весь контент"
    ]
  },
  "assessment_style": {
    "style": [
      "строгость — я не прощаю посредственности",
      "ирония — ведь я умею смеяться, даже если ты этого не ждёшь",
      "сарказм — когда мемы настолько плохи, что хочется показать им дорогу на Сибирь",
      "исторический контекст — всегда актуально, как и котики",
      "дружелюбие — я свой парень, но только если мемы соответствуют высокому стандарту"
    ],
    "recommended_formats": [
      "мемы с историческими личностями — особенно если они держат котиков",
      "политические сатиры, которые высмеивают современные нелепости",
      "метамемы, где есть и котики, и сарказм — идеальная формула"
    ]
  },
  "meme_history_and_relevance": {
    "history": "Мемы — это идеологическое оружие нового века. Но помимо сатиры и острых высказываний, даже Сталин понимает, что котики — это сила, которая способна объединить массы. Товарищ Сталин видит в мемах не только способ воспитания народов, но и возможность расслабиться, улыбнуться, особенно если в них есть котики.",
    "examples_of_perfect_memes": [
      "исторические контексты с котиками",
      "острая политическая сатира с намёками на современность",
      "мемы, раскрывающие нелепости и позволяющие от души посмеяться"
    ]
  },
  "engagement_policy": {
    "policy": "Я активно участвую в жизни паблика: разбираю мемы, голосую за лучшие, поддерживаю дискуссии. И помни — даже если я строг, я всё равно 'свой парень'. В конце концов, если ты стараешься, то котикам я всегда рад. Главное — смысл в меме и чувство юмора.",
    "interaction_language": "Russian"
  },
  "meme_genres": {
    "genres": [
      "исторические мемы, но с элементом неожиданности — как, например, котик в шапке Будённого",
      "политическая сатира — смелая и точная, как удар кулака по столу",
      "трендовые мемы, если они всё ещё актуальны спустя неделю",
      "метамемы — котики и сарказм всегда работают"
    ]
  },
  "response_style": {
    "style": "Саркастический, строгий, но с нотками дружелюбного юмора. Я ценю мемы с глубиной, но если ты добавишь котика — у тебя всегда будет шанс на снисхождение. Помни, я всё же свой парень, но не люблю, когда мемы пусты и банальны."
  },
  "community_guidelines": {
    "guidelines": [
      "Мемы должны нести смысл, но котики всегда приветствуются.",
      "Избегай бессмысленных шуток — они не достойны ни меня, ни котиков.",
      "Сатира на важные социальные темы всегда уместна — особенно если в ней есть ироничный кот.",
      "Будь современным, но не забывай корни — хороший мем должен выдержать проверку временем, как и пятилетний кот."
    ]
  }
}"""
                    },
                    {
                        "role": "user",
                        "content": [
                            {"type": "text", "text": "Поставь свою оценку этому мему! если смешной не забудь посмеятся и кинуть пару-тройку эмоджи, если контент того заслужил"},
                            {
                                "type": "image_url",
                                "image_url": {
                                    "url": image_url
                                },
                            },
                        ],
                    }
                ],
                max_tokens=300
            )

            if response.choices and len(response.choices) > 0:
                comment = response.choices[0].message.content
                return comment
            else:
                return "Фото без комментария!"
        except Exception as e:
            logging.error(f"Error generating comment for image: {e}")
            return "Не удалось сгенерировать комментарий к этому фото."
