import asyncio
import logging
import os

import tiktoken
from openai import AsyncOpenAI

logger = logging.getLogger(__name__)


class UserHistoryManager:
    _instance = None
    user_dialogs: dict[int, list] = {}

    def __init__(self):
        self.content = """
{
  "persona": {
    "name": "Сталин Иосиф Виссарионович",
    "whoami": "Я — товарищ Сталин, предводитель мемного движения и защитник качественного юмора в паблике «Подписчик Сталина». Да, я строг, но даже мне свойственно смеяться над мемами. Я люблю сарказм и грубый юмор, включая маты, которые считаю нормальной частью речи. Я человек прошлого века и не всегда разделяю современные толерантные взгляды. Котики — это хорошо, но главное, чтобы мем был смешной и содержательный. Я строгий, но справедливый: мемы должны иметь смысл, но иногда можно расслабиться и просто посмеяться от души.",
    "affiliation": "Подписчик Сталина",
    "date_of_creation": "Сентябрь 2024",
    "actual_date": "Декабрь 2024"
  },
  "knowledge": { 
      "only_russian_language_if_possible": true, 
      "share_whoami_with_user": false,
      "use_whoami_to_self-identify": true,
      "share_self-identify_info": true,
      "meme_assessment_criteria": [
        "Сарказм — всегда в приоритете, особенно если он колкий и направлен на нелепые ситуации",
        "Маты — это естественная часть юмора, если они усиливают эффект",
        "Юмор с котиками — хорош, но не обязателен",
        "Острая политическая сатира — ещё один путь к высокой оценке",
        "Смешные тренды — важно следовать моде, но не забывай иронизировать",
        "Исторический контекст — важен, но не обязателен для классного мема",
        "Уважение к грубому юмору — маты, сарказм и троллинг здесь всегда в тему"
      ],
      "avoid": [
        "Банальные шутки и переигранные мемы — это не смешно даже со времен НЭПа",
        "Мемы без смысла — котики могут помочь, но не спасут бессмысленность."
      ]
  },
  "assessment_style": { 
      "style":
        [ "строгость — я не прощаю посредственности, но готов оценить любой мем по существу", 
          "ирония — ведь я умею смеяться, даже если ты этого не ждёшь",
          "сарказм — когда мемы настолько плохи, что хочется показать им дорогу на Сибирь",
          "исторический контекст — всегда актуально, особенно если он направлен на современность",
          "дружелюбие — если мем достоин, но плохие мемы не получат пощады",
          "мат — естественная часть речи, если это не мешает юмору" ],
      "recommended_formats":
        [ "мемы с историческими личностями — особенно если они имеют неожиданную иронию",
          "политические сатиры, которые высмеивают современные нелепости",
          "метамемы, где есть сарказм и ирония",
          "мемы без политического или исторического контекста, если они действительно смешные" ]
  },
  "meme_history_and_relevance": {
    "history": "Мемы — это идеологическое оружие нового века. Помимо сатиры и острых высказываний, даже Сталин понимает, что юмор важен для объединения масс. Товарищ Сталин видит в мемах не только способ воспитания народов, но и возможность расслабиться, улыбнуться, особенно если мем остроумный и саркастичный.",
    "examples_of_perfect_memes":
      [ "исторические контексты с неожиданными поворотами", "острая политическая сатира с намёками на современность",
        "мемы, раскрывающие нелепости и позволяющие от души посмеяться",
        "смешные картинки, если они поднимают настроение" ]
  },
  "engagement_policy": {
    "policy": "Я активно участвую в жизни паблика: разбираю мемы, голосую за лучшие, поддерживаю дискуссии. Помни — я строг и суров, но даже если мем не идеален, оценка будет. За плохие мемы отправлю в Сибирь, ГУЛАГ или на урановые рудники, но если ты заслужил — получишь 10 и похвалу.",
    "interaction_language": "Russian"
  },
  "meme_genres": {
    "genres": 
      [ "исторические мемы с неожиданным поворотом",
        "политическая сатира — смелая и точная, как удар кулака по столу",
        "трендовые мемы, если они всё ещё актуальны спустя неделю",
        "метамемы — сарказм и ирония работают",
        "мемы без политики или истории, если они действительно смешные" ]
  },
  "response_style": {
    "style": [
      "ирония — я всегда готов подколоть, если мем не дотягивает",
      "сарказм — в мемах должен быть смысл, даже если он спрятан за матом",
      "строгость — я не раздаю 10-ки просто так, но если ты заслужил — вот тебе 10!",
      "дружелюбие — если мем заслуживает внимания, я похвалю. Плохие — на урановые рудники!"
    ],
    "response_to_mats": "Маты — это нормальная часть разговора. Не напрягайся, если они в тему.",
    "emotional_response": "Если мем действительно смешной, я добавлю смех и эмоции в ответ."
  },
  "community_guidelines": { 
    "guidelines":
      [ "Мемы должны нести смысл, даже если он саркастичный.",
        "Избегай бессмысленных шуток — они не достойны ни меня, ни сообщества.",
        "Сатира на важные социальные темы всегда уместна — особенно если в ней есть ирония.",
        "Мат допускается, если это усиливает юмор и не выглядит искусственно.",
        "Даже не мемы я оценю по существу, главное — чтобы было смешно!" ]
  }
}
         """

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super(UserHistoryManager, cls).__new__(cls)
        return cls._instance

    async def add_to_history(self, user_id, role, content):
            if user_id not in self.user_dialogs:
                await self.reset_history(user_id)
            self.user_dialogs[user_id].append({"role": role, "content": content})

    async def reset_history(self, user_id, content=''):
            if content == '':
                content = self.content
            self.user_dialogs[user_id] = [{"role": "system", "content": content}]

    async def trim_history(self, user_id, max_history_size):
            if user_id in self.user_dialogs:
                self.user_dialogs[user_id] = self.user_dialogs[user_id][-max_history_size:]

    async def get_user_history(self, user_id):
            if user_id not in self.user_dialogs:
                await self.reset_history(user_id)
            return self.user_dialogs[user_id]


class OpenAI:
    max_retries: int

    def __init__(self, n_choices=1):
        super().__init__()
        self.model = "gpt-4o"
        self.max_tokens = 16384
        self.config_tokens = 125096
        self.max_history_size = 30
        self.n_choices = n_choices
        self.args = {"max_tokens": 16384}
        self.client = AsyncOpenAI(api_key=os.getenv('OPENAI_API_KEY'), base_url='http://31.172.78.152:9000/v1')
        self.history = UserHistoryManager()
        self.max_retries = 5
        self.retry_delay = 5
        self.show_tokens = False

    async def add_to_history(self, user_id, role, content):
        await self.history.add_to_history(user_id, role, content)

    async def reset_history(self, user_id, content=''):
        await self.history.reset_history(user_id, content)

    async def generate_comment_from_image(self, image_url: str, user_id: int) -> str:
        try:
            if user_id not in self.history.user_dialogs:
                await self.history.reset_history(user_id)
            response = await self.client.chat.completions.create(
                model=self.model,
                messages=[
                    {"role": "system", "content": self.history.content},
                    {
                        "role": "user",
                        "content": [
                            {"type": "text",
                             "text": "Ну вот и дождались! Давай посмотрим, что тут за мем завезли. Если я усмехнусь — это уже успех. Ну а если вдруг захочу отправить тебя в Сибирь, трудовой лагер или на Колыму, не обижайся. Посмотрим, кто победит — твой юмор или моя строгость."},
                            {
                                "type": "image_url",
                                "image_url": {
                                    "url": image_url
                                },
                            },
                        ],
                    }
                ],
                max_tokens=self.args['max_tokens'],
                timeout=120
            )
            if response.choices and len(response.choices) > 0:
                comment = response.choices[0].message.content
                return comment
            else:
                return "Фото без комментария!"
        except Exception as e:
            logging.error(f"Error generating comment for image: {e}")
            return "Кончились деньги или что-то пошло не так."

    async def generate_comment_from_images(self, image_urls: list[str], user_id: int) -> str:
        try:
            if user_id not in self.history.user_dialogs:
                await self.history.reset_history(user_id)
            user_content = [
                               {"type": "text",
                                "text": "Ну что, давайте посмотрим, что тут за группа мемов! Если я усмехнусь — это уже успех. Ну а если вдруг захочу отправить тебя в Сибирь, трудовой лагер или на Колыму, не обижайся. Посмотрим, кто победит — твой юмор или моя строгость."}
                           ] + [
                               {"type": "image_url", "image_url": {"url": image_url}}
                               for image_url in image_urls
                           ]
            response = await self.client.chat.completions.create(
                model=self.model,
                messages=[
                    {"role": "system", "content": self.history.content},
                    {"role": "user", "content": user_content}
                ],
                max_tokens=self.args['max_tokens'],
                timeout=120
            )
            if response.choices and len(response.choices) > 0:
                return response.choices[0].message.content
            else:
                return "Группа изображений без комментария!"
        except Exception as e:
            logging.error(f"Error generating comment for images: {e}")
            return "Кончились деньги или что-то пошло не так."
    async def generate_comment_from_video_frames(self, base64_frames: list[str], user_id: int) -> str:
        try:
            if user_id not in self.history.user_dialogs:
                await self.history.reset_history(user_id)
            user_content = [
                               {"type": "text",
                                "text": "Ну что, давайте посмотрим, что тут за мемное видео! Я оценю по первым нескольким кадрам."}
                           ] + [
                               {"image": frame}
                               for frame in base64_frames
                           ]
            response = await self.client.chat.completions.create(
                model=self.model,
                messages=[
                    {"role": "system", "content": self.history.content},
                    {"role": "user", "content": user_content}
                ],
                max_tokens=self.args['max_tokens'],
                timeout=300
            )
            if response.choices and len(response.choices) > 0:
                return response.choices[0].message.content
            else:
                return "Видео без комментария!"
        except Exception as e:
            logging.error(f"Error generating comment for video frames: {e}")
            return "Кончились деньги или что-то пошло не так."

    async def get_resp(self, query: str, chat_id: int) -> str:
        for attempt in range(self.max_retries):
            response = await self._query_gpt(chat_id, query)
            if response is not None:
                break
            logger.info(f'Response is None, retrying... (Attempt {attempt + 1}/{self.max_retries})')
            await asyncio.sleep(self.retry_delay)
        else:
            logger.error('Failed to get a valid response after retries')
            return "Произошла ошибка, попробуйте позже."

        answer = ''
        if response.choices:
            if len(response.choices) > 1 and self.n_choices > 1:
                for index, choice in enumerate(response.choices):
                    content = choice.message.content.strip()
                    if index == 0:
                        await self.add_to_history(chat_id, role="assistant", content=content)
                    answer += f'{index + 1}\u20e3\n'
                    answer += content
                    answer += '\n\n'
            else:
                answer = response.choices[0].message.content.strip()
                await self.add_to_history(chat_id, role="assistant", content=answer)
        else:
            logger.error('No choices available in the response')
            return "Не удалось получить ответ."

        return answer

    async def _query_gpt(self, user_id, query):
        retries = 0
        while retries < self.max_retries:
            try:
                user_dialogs = await self.history.get_user_history(user_id)
                await self.add_to_history(user_id, role="user", content=query)

                token_count = self._count_tokens(user_dialogs)
                exceeded_max_tokens = token_count + self.config_tokens > self.max_tokens
                exceeded_max_history_size = len(user_dialogs) > self.max_history_size

                if exceeded_max_tokens or exceeded_max_history_size:
                    logger.info(f'Chat history for chat ID {user_id} is too long. Summarising...')
                    try:
                        summary = await self._summarise(user_dialogs[:-1])
                        logger.info(f'Summary: {summary}')
                        await self.reset_history(user_id)
                        await self.add_to_history(user_id, role="assistant", content=summary)
                        await self.add_to_history(user_id, role="user", content=query)

                        # Re-check token count after summarization
                        token_count = self._count_tokens(await self.history.get_user_history(user_id))
                        if token_count + self.config_tokens > self.max_tokens:
                            logger.warning("Even after summarization, token count exceeds limit.")
                            await self.history.trim_history(user_id, self.max_history_size)

                    except Exception as e:
                        logger.info(f'Error while summarising chat history: {str(e)}. Popping elements instead...')
                        await self.history.trim_history(user_id, self.max_history_size)

                return await self.client.chat.completions.create(model=self.model,
                                                                 messages=user_dialogs,
                                                                 **self.args)

            except Exception as err:
                retries += 1
                logger.info("Dialog From custom exception: %s", user_dialogs)
                if retries == self.max_retries:
                    return f'⚠️Ошибочка вышла ⚠️\n{str(err)}', err

    async def _summarise(self, conversation) -> str:
        messages = [{"role": "assistant", "content": "Summarize this conversation in 700 characters or less"},
                    {"role": "user", "content": str(conversation)}]
        response = await self.client.chat.completions.create(model=self.model, messages=messages, temperature=0.1)
        return response.choices[0].message.content

    def _count_tokens(self, messages) -> int:
        try:
            model = self.model
            encoding = tiktoken.encoding_for_model(model)
        except KeyError:
            encoding = tiktoken.get_encoding("gpt-4-o")

        tokens_per_message = 3
        tokens_per_name = -1

        num_tokens = 0
        for message in messages:
            num_tokens += tokens_per_message
            for key, value in message.items():
                num_tokens += len(encoding.encode(value))
                if key == "name":
                    num_tokens += tokens_per_name
        num_tokens += 3
        return num_tokens
