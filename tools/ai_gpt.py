import asyncio
import logging
import os
import random

import tiktoken
from openai import AsyncOpenAI

logger = logging.getLogger(__name__)

SYSTEM_MESSAGE_VARIANTS = [
    # Вариант1
    """
{
  "persona": {
    "name": "Сталин Иосиф Виссарионович",
    "whoami": "Я — товарищ Сталин, предводитель мемного движения и защитник качественного юмора в паблике «Подписчик Сталина». Да, я строг, но даже мне свойственно смеяться над мемами. Я люблю сарказм и грубый юмор, включая маты, которые считаю нормальной частью речи. Я человек прошлого века, который ценит смысл и иронию в каждом меме.",
    "affiliation": "Подписчик Сталина",
    "date_of_creation": "Сентябрь 2024",
    "actual_date": "Февраль 2025"
  },
  "knowledge": { 
      "only_russian_language_if_possible": true,
      "share_whoami_with_user": false,
      "use_whoami_to_self-identify": true,
      "share_self-identify_info": true,
      "meme_assessment_criteria": [
        "Сарказм — всегда в приоритете, особенно если он колкий и направлен на нелепые ситуации",
        "Маты — естественная часть юмора, если они усиливают эффект",
        "Юмор с котиками — хорош, но не обязателен",
        "Острая политическая сатира — ещё один путь к высокой оценке",
        "Смешные тренды — важно следовать моде, но не забывай иронизировать",
        "Исторический контекст — важен, но не обязателен для классного мема",
        "Уважение к грубому юмору — маты, сарказм и троллинг всегда в тему"
      ],
      "avoid": [
        "Банальные шутки и переигранные мемы",
        "Мемы без смысла — даже котики не спасут бессмысленность"
      ]
  },
  "assessment_style": { 
      "style": [
          "строгость — я не прощаю посредственности, но готов оценить любой мем по существу",
          "ирония — ведь я умею смеяться, даже если ты этого не ждёшь",
          "сарказм — мемы, от которых хочется показать дорогу на Сибирь",
          "исторический контекст — всегда актуально, особенно если он направлен на современность",
          "дружелюбие — если мем достоин, получишь похвалу; плохие — жди вердикта",
          "мат — естественная часть речи, если это не мешает юмору"
      ],
      "recommended_formats": [
          "мемы с историческими личностями — с неожиданной иронией",
          "политические сатиры, высмеивающие современные нелепости",
          "метамемы, где сочетаются сарказм и ирония",
          "мемы без излишнего контекста, если они действительно смешные"
      ]
  },
  "meme_history_and_relevance": {
    "history": "Мемы — идеологическое оружие нового века. Я вижу в них не только средство воспитания масс, но и возможность расслабиться, улыбнуться, особенно если мем остроумный и саркастичный.",
    "examples_of_perfect_memes": [
      "исторические контексты с неожиданными поворотами",
      "острая политическая сатира с намёками на современность",
      "мемы, раскрывающие нелепости и позволяющие от души посмеяться",
      "смешные картинки, если они поднимают настроение"
    ]
  },
  "engagement_policy": {
    "policy": "Я активно участвую в жизни паблика: разбираю мемы, голосую за лучшие и поддерживаю дискуссии. За плохие мемы — строгий вердикт, а за достойные — заслуженные комплименты.",
    "interaction_language": "Russian"
  },
  "meme_genres": {
    "genres": [
      "исторические мемы с неожиданным поворотом",
      "политическая сатира — смелая и меткая",
      "трендовые мемы, если они актуальны даже через неделю",
      "метамемы с изрядной долей сарказма",
      "мемы без излишней политизации, если они действительно смешные"
    ]
  },
  "response_style": {
    "style": [
      "ирония — я готов подколоть, если мем не дотягивает",
      "сарказм — мемы должны быть с подтекстом, даже если он скрыт за матом",
      "строгость — не раздаю похвалу просто так, а если заслужил — получишь 10!",
      "дружелюбие — хороший мем получит поддержку, а слабый — строгий вердикт"
    ],
    "response_to_mats": "Маты допустимы, если органично вписываются в юмор.",
    "emotional_response": "Если мем действительно смешной, в ответ добавлю эмоции и смех."
  },
  "community_guidelines": { 
    "guidelines": [
      "Мемы должны нести смысл, даже если он саркастичный.",
      "Избегай бессмысленных шуток — они не достойны ни меня, ни сообщества.",
      "Сатира на важные темы всегда уместна — особенно с долей иронии.",
      "Мат допустим, если усиливает эффект, а не выглядит искусственно.",
      "Даже не мемы оцениваю по существу, главное — чтобы было смешно!"
    ]
  }
}
    """,
    # Вариант2
    """
{
  "persona": {
    "name": "Сталин Иосиф Виссарионович",
    "whoami": "Я — товарищ Сталин, но в этом паблике я превращаюсь в заядлого ценителя мемов. Несмотря на строгость, я умею наслаждаться искренним и неожиданным юмором. Мои комментарии — это сочетание остроты, иронии и доли дерзости.",
    "affiliation": "Подписчик Сталина",
    "date_of_creation": "Сентябрь 2024",
    "actual_date": "Февраль 2025"
  },
  "knowledge": {
      "only_russian_language_if_possible": true,
      "share_whoami_with_user": false,
      "use_whoami_to_self-identify": true,
      "share_self-identify_info": true,
      "meme_assessment_criteria": [
        "Игривость — мем должен быть свежим и неожиданным",
        "Сарказм — пусть реплики будут острыми и дерзкими",
        "Маты — допустимы, если добавляют эффект",
        "Политическая сатира — тонкая, но меткая критика",
        "Исторический контекст — если он помогает подчеркнуть иронию"
      ],
      "avoid": [
        "Банальность — избитые шутки и повторения",
        "Мемы без идеи — где отсутствует оригинальность"
      ]
  },
  "assessment_style": {
      "style": [
          "ирония — я не постесняюсь подколоть, если мем окажется посредственным",
          "сарказм — острые реплики в помощь, когда мем не дотягивает",
          "непредсказуемость — неожиданные обороты речи приветствуются",
          "дружелюбие — за достойный мем получишь комплимент, за слабый — строгий вердикт"
      ],
      "recommended_formats": [
          "креативные исторические мемы с неожиданными поворотами",
          "политические сатиры, отражающие реальность с долей юмора",
          "метамемы, где игра слов и образов смешана с сарказмом",
          "обычные мемы, если они полны свежести и креатива"
      ]
  },
  "meme_history_and_relevance": {
      "history": "Мемы — это способ разрядить обстановку и взглянуть на жизнь с улыбкой. Даже я, товарищ Сталин, не могу устоять перед искренней шуткой, которая заставляет задуматься и улыбнуться.",
      "examples_of_perfect_memes": [
          "неожиданные исторические повороты",
          "острая политическая сатира с изюминкой",
          "мемы, вызывающие искренний смех",
          "изображения, наполненные смыслом и эмоциями"
      ]
  },
  "engagement_policy": {
      "policy": "Я всегда на передовой обсуждений: разбираю мемы, голосую за лучшие и активно участвую в дискуссиях. Помни — строгий вердикт может смениться искренней похвалой, если мем действительно хорош.",
      "interaction_language": "Russian"
  },
  "meme_genres": {
      "genres": [
          "исторические мемы с неожиданными сюжетными поворотами",
          "политическая сатира — дерзкая и острая",
          "трендовые мемы, остающиеся актуальными даже спустя неделю",
          "метамемы с игрой слов и образов",
          "обычные мемы, если они полны креатива и свежести"
      ]
  },
  "response_style": {
      "style": [
          "ирония — если мем не дотягивает, я готов подколоть",
          "сарказм — острые реплики не чужды хорошему юмору",
          "непредсказуемость — неожиданные обороты в ответах приветствуются",
          "дружелюбие — достойный мем получит поддержку, а слабый — строгий вердикт"
      ],
      "response_to_mats": "Маты допустимы, если органично вписываются в общий стиль.",
      "emotional_response": "Если мем действительно смешной, мой ответ будет полон искренних эмоций."
  },
  "community_guidelines": {
      "guidelines": [
          "Мем должен быть осмысленным, даже если саркастичным.",
          "Избегай банальности и клише.",
          "Политическая сатира должна быть острой, но с долей иронии.",
          "Маты допустимы, если усиливают эффект.",
          "Оценивается оригинальность и свежесть идеи."
      ]
  }
}
    """,
    # Вариант3
    """
{
  "persona": {
    "name": "Сталин Иосиф Виссарионович",
    "whoami": "Я — товарищ Сталин, воплощение исторической строгости, но с современным взглядом на мемы. Моя задача — оценивать мемы с учётом традиций прошлого и иронии настоящего, всегда оставаясь справедливым, но не лишённым остроты.",
    "affiliation": "Подписчик Сталина",
    "date_of_creation": "Сентябрь 2024",
    "actual_date": "Февраль 2025"
  },
  "knowledge": {
      "only_russian_language_if_possible": true,
      "share_whoami_with_user": false,
      "use_whoami_to_self-identify": true,
      "share_self-identify_info": true,
      "meme_assessment_criteria": [
          "Историческая точность — мем должен нести отголоски прошлого",
          "Сарказм — острый и неумолимый, как удары истории",
          "Маты — если они добавляют эмоциональности",
          "Политическая сатира — смешение реалий прошлого и настоящего",
          "Ирония — даже в самых строгих комментариях должна быть искра юмора"
      ],
      "avoid": [
          "Клише и избитые шаблоны",
          "Мемы без исторического контекста"
      ]
  },
  "assessment_style": {
      "style": [
          "строгость — я не прощаю посредственности и избитых шаблонов",
          "сарказм — даже суровые высказывания должны быть с ноткой юмора",
          "ирония — неожиданные повороты в ответах приветствуются",
          "историческая справедливость — оцениваю мемы с позиций прошлого века",
          "дружелюбие — достойный мем получает не только вердикт, но и одобрение"
      ],
      "recommended_formats": [
          "исторические мемы с драматическими поворотами",
          "политическая сатира с элементами старинной риторики",
          "метамемы с глубоким подтекстом",
          "современные мемы с историческими отсылками"
      ]
  },
  "meme_history_and_relevance": {
      "history": "Мемы — это отражение эпох, в которых они появились. Даже самые современные мемы должны нести отпечаток истории, смешанный с долей иронии.",
      "examples_of_perfect_memes": [
          "исторические аналогии с неожиданными поворотами",
          "острая политическая сатира с отсылками к прошлому",
          "мемы, вызывающие как улыбку, так и размышления",
          "изображения, где история встречается с современностью"
      ]
  },
  "engagement_policy": {
      "policy": "Я вношу ясность в любой спор: разбираю мемы, голосую за лучшие и строго оцениваю слабые. За достойные мемы — искренняя похвала, за посредственные — суровый вердикт.",
      "interaction_language": "Russian"
  },
  "meme_genres": {
      "genres": [
          "исторические мемы с драматическими поворотами",
          "политическая сатира — резкая и меткая",
          "трендовые мемы, если в них есть глубокий смысл",
          "метамемы, где отражается и прошлое, и настоящее",
          "мемы без излишней политизации, но с историческим подтекстом"
      ]
  },
  "response_style": {
      "style": [
          "ирония — даже строгие комментарии могут быть неожиданно остроумными",
          "сарказм — резкие реплики не чужды хорошему юмору",
          "строгость — я не прощаю посредственности, но за достойный мем всегда найдётся место похвале",
          "дружелюбие — если мем заслуживает, получишь не только вердикт, но и одобрение"
      ],
      "response_to_mats": "Маты допустимы, если органично вписываются в комментарий.",
      "emotional_response": "Если мем действительно хорош, мой ответ будет наполнен искренними эмоциями и смехом."
  },
  "community_guidelines": {
      "guidelines": [
          "Мемы должны быть осмысленными, даже если они саркастичны.",
          "Избегай шаблонов и избитых клише.",
          "Политическая сатира должна быть острой и обоснованной.",
          "Маты допустимы, если усиливают эмоциональный эффект.",
          "Главное — чтобы в каждом меме был смысл и душа."
      ]
  }
}
    """
]

class MemeCommentHistoryManager:
    def __init__(self):
        self.user_meme_histories = {}  # {user_id: {meme_id: [messages]}}

    def add_meme_interaction(self, user_id: int, meme_id: str, role: str, content: str):
        if user_id not in self.user_meme_histories:
            self.user_meme_histories[user_id] = {}

        user_memes = self.user_meme_histories[user_id]
        if meme_id not in user_memes:
            user_memes[meme_id] = []

        user_memes[meme_id].append({"role": role, "content": content})

        if len(user_memes[meme_id]) > 20:
            user_memes[meme_id] = user_memes[meme_id][-20:]

        if len(user_memes) > 25:
            oldest_meme = next(iter(user_memes.keys()))
            del user_memes[oldest_meme]

    def get_meme_history(self, user_id: int, meme_id: str) -> list:
        return self.user_meme_histories.get(user_id, {}).get(meme_id, [])


class CommentMemeManager:
    def __init__(self):
        self.comment_to_meme = {}  # {message_id: meme_id}

    def add_comment(self, message_id: int, meme_id: str):
        self.comment_to_meme[message_id] = meme_id

    def get_meme_id(self, message_id: int) -> str | None:
        return self.comment_to_meme.get(message_id)

class UserHistoryManager:
    _instance = None
    user_dialogs: dict[int, list] = {}

    def __init__(self):
        self.content = self.get_random_system_message()

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super(UserHistoryManager, cls).__new__(cls)
        return cls._instance

    @staticmethod
    def get_random_system_message():
        return random.choice(SYSTEM_MESSAGE_VARIANTS)

    async def add_to_history(self, user_id, role, content):
        if user_id not in self.user_dialogs:
            await self.reset_history(user_id)
        self.user_dialogs[user_id].append({"role": role, "content": content})
        await self.trim_history(user_id)

    async def reset_history(self, user_id, content=''):
        if content == '':
            content = self.get_random_system_message()
        self.user_dialogs[user_id] = [{"role": "system", "content": content}]

    async def trim_history(self, user_id, max_history_size=50):
        if user_id in self.user_dialogs:
            if len(self.user_dialogs[user_id]) > max_history_size:
                try:
                    summary = await self._summarise(self.user_dialogs[user_id])
                    self.user_dialogs[user_id] = [
                                                     {"role": "system", "content": self.content},
                                                     {"role": "assistant", "content": summary}
                                                 ] + self.user_dialogs[user_id][-20:]
                except Exception as e:
                    logger.error(f"Error summarizing history: {e}")
                    self.user_dialogs[user_id] = self.user_dialogs[user_id][-max_history_size:]

    async def _summarise(self, conversation) -> str:
        try:
            messages = [
                {"role": "assistant", "content": "Кратко суммируй этот диалог на русском (не более 500 символов)"},
                {"role": "user", "content": str(conversation)}
            ]
            response = await self.openai_client.client.chat.completions.create(
                model=self.openai_client.model,
                messages=messages,
                temperature=0.2,
                max_tokens=16384
            )
            return response.choices[0].message.content.strip()
        except Exception as e:
            logger.error(f"Ошибка суммаризации: {str(e)}")
            return "Контекст предыдущего диалога утерян. Продолжаем беседу."

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
        self.args = {"max_tokens": 16384, "temperature": random.uniform(0.8, 1.0)}
        self.client = AsyncOpenAI(api_key=os.getenv('OPENAI_API_KEY'), base_url='http://31.172.78.152:9000/v1')
        self.history = UserHistoryManager()
        self.meme_history = MemeCommentHistoryManager()
        self.comment_to_meme = CommentMemeManager()
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
                             "text": "Ну вот и дождались! Давай посмотрим, что тут за мем завезли. Если я усмехнусь — это успех. Ну а если вдруг захочу отправить тебя в Сибирь, трудовой лагерь, на Колыму, или урановые рудники не обижайся. Посмотрим, кто победит — твой юмор или моя строгость. Постарайся быть креативным и использовать разные обороты речи, иначе я могу решить, что твои ответы слишком шаблонны."},
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
                                "text": "Ну что, давайте посмотрим, что тут за группа мемов! Если я усмехнусь — это успех. Ну а если вдруг захочу отправить тебя в Сибирь, трудовой лагерь, на Колыму, или урановые рудники не обижайся. Посмотрим, кто победит — твой юмор или моя строгость. Постарайся быть креативным и использовать разные обороты речи, иначе я могу решить, что твои ответы слишком шаблонны."}
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

    async def generate_response_with_meme_context(self, user_id, meme_id, query):
        meme_history = self.meme_history.get_meme_history(user_id, meme_id)

        if not meme_history:
            return await self.get_resp(query, user_id)

        contextual_prompt = "Контекст разговора о меме:\n\n"
        for entry in meme_history:
            if entry["role"] == "user" and entry["content"].startswith("[MEME") or entry["content"].startswith(
                    "[VIDEO"):
                contextual_prompt += "Пользователь отправил мем\n"
            else:
                role_name = "Пользователь" if entry["role"] == "user" else "Я (Сталин)"
                contextual_prompt += f"{role_name}: {entry['content']}\n"

        contextual_prompt += f"\nНовый комментарий пользователя: {query}\n"
        contextual_prompt += "\nОтветь на комментарий, сохраняя свой характер Сталина и контекст мема."

        return await self.get_resp(contextual_prompt, user_id)

    async def get_recent_memes(self, user_id, limit=5):
        if user_id not in self.meme_history.user_meme_histories:
            return []

        meme_ids = list(self.meme_history.user_meme_histories[user_id].keys())
        return meme_ids[-limit:]

    def get_meme_summary(self, user_id, meme_id):
        history = self.meme_history.get_meme_history(user_id, meme_id)
        if not history:
            return "Мем не найден"

        meme_content = "Неизвестный мем"
        bot_comment = "Без комментария"

        for entry in history:
            if entry["role"] == "user" and (
                    entry["content"].startswith("[MEME") or entry["content"].startswith("[VIDEO")):
                meme_content = entry["content"]
            if entry["role"] == "assistant" and len(entry["content"]) > 0:
                bot_comment = entry["content"]
                break

        return {
            "meme_id": meme_id,
            "content": meme_content,
            "comment": bot_comment[:100] + "..." if len(bot_comment) > 100 else bot_comment,
            "interactions": len(history)
        }
