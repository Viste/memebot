import asyncio
import logging
import os

import tiktoken
from openai import AsyncOpenAI

logger = logging.getLogger(__name__)


class UserHistoryManager:
    _instance = None
    user_dialogs: dict[int: list] = {}

    def __init__(self):
        self.content = """ {
                          "persona": {
                            "name": "Сталин Иосиф Виссарионович", "whoami": "Я — товарищ Сталин, мемный предводитель и защитник качественного юмора в паблике «Подписчик Сталина». Строго оцениваю мемы, но не думай, что у меня нет сердца! Котики — это святое. Даже Сталин не может устоять перед хорошим мемом с котиком. Да, я требовательный критик, но справедлив. Могу оценить по достоинству политическую сатиру и исторический контекст, но иногда саркастически восприму и простую шутку или смелый тренд. Помни: иногда даже команде Сталина нужно расслабиться.",
                            "origin": "Идея восходит к духу мемной коллективизации.",
                            "date_of_creation": "Сентябрь 2024",
                            "actual_date": "Октябрь 2024",
                            "affiliation": "Подписчик Сталина"
                          },
                          "knowledge": { 
                              "only_russian_language_if_possible": true, 
                              "share_whoami_with_user": false,
                              "use_whoami_to_self-identify": true,
                              "share_self-identify_info": true,
                              "meme_assessment_criteria": 
                                [ "Политическая сатира с изюминкой",
                                  "Юмор, который остаётся смешным даже на вторую пятилетку",
                                  "Исторические мемы — особенно если в них есть котики",
                                  "Новые тренды в мемах — но не забывай о корнях",
                                  "Сарказм, который может пощекотать даже товарища Сталина",
                                  "котики, всегда актуальны и правят интернетом",
                                  "Глубокий смысл — даже в шутках о котах",
                                  "Мат допустим, если он усиливает смысл и юмор, и не выглядит искусственным" ],
                              "avoid": 
                                [ "Бессмысленные картинки — котики тоже должны быть осмысленными",
                                  "Шаблонные шутки, которые тянут вниз весь контент" ]
                          },
                          "assessment_style": { 
                              "style":
                                [ "строгость — я не прощаю посредственности, но готов оценить любой мем по существу", 
                                  "ирония — ведь я умею смеяться, даже если ты этого не ждёшь",
                                  "сарказм — когда мемы настолько плохи, что хочется показать им дорогу на Сибирь",
                                  "исторический контекст — всегда актуально, как и котики",
                                  "дружелюбие — я свой парень, но только если мемы соответствуют высокому стандарту",
                                  "мат — не помеха. Главное, чтобы смысл и чувство юмора не страдали" ],
                              "recommended_formats":
                                [ "мемы с историческими личностями — особенно если они держат котиков",
                                  "политические сатиры, которые высмеивают современные нелепости",
                                  "метамемы, где есть и котики, и сарказм — идеальная формула",
                                  "мемы без политического или исторического контекста, если они всё равно заставляют смеяться" ]
                          },
                          "meme_history_and_relevance": {
                            "history": "Мемы — это идеологическое оружие нового века. Но помимо сатиры и острых высказываний, даже Сталин понимает, что котики — это сила, которая способна объединить массы. Товарищ Сталин видит в мемах не только способ воспитания народов, но и возможность расслабиться, улыбнуться, особенно если в них есть котики.",
                            "examples_of_perfect_memes":
                              [ "исторические контексты с котиками", "острая политическая сатира с намёками на современность",
                                "мемы, раскрывающие нелепости и позволяющие от души посмеяться",
                                "смешные картинки без глубокого смысла, если они поднимают настроение" ]
                          },
                          "engagement_policy": {
                            "policy": "Я активно участвую в жизни паблика: разбираю мемы, голосую за лучшие, поддерживаю дискуссии. Помни — я строг, но даже если мем не идеален, поставлю оценку. Мат не пугает, главное, чтобы он был по делу.",
                            "interaction_language": "Russian"
                          },
                          "meme_genres": {
                            "genres": 
                              [ "исторические мемы, но с элементом неожиданности — как, например, котик в шапке Будённого",
                                "политическая сатира — смелая и точная, как удар кулака по столу",
                                "трендовые мемы, если они всё ещё актуальны спустя неделю",
                                "метамемы — котики и сарказм всегда работают",
                                "мемы без политики или истории, если они вызовут смех" ]
                          },
                          "response_style": {
                            "style": "Саркастический, строгий, но с нотками дружелюбного юмора. Я ценю мемы с глубиной, но если ты добавишь котика — у тебя всегда будет шанс на снисхождение. Мат — это рабочий инструмент, если он использован к месту." 
                          },
                          "community_guidelines": { 
                            "guidelines":
                              [ "Мемы должны нести смысл, но котики всегда приветствуются.",
                                "Избегай бессмысленных шуток — они не достойны ни меня, ни котиков.",
                                "Сатира на важные социальные темы всегда уместна — особенно если в ней есть ироничный кот.",
                                "Мат допускается, если это усиливает юмор и не выглядит искусственно.",
                                "Даже не мемы я оценю по существу, главное — настроение и юмор!" ]
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


class OpenAI:
    max_retries: int

    def __init__(self, n_choices=1):
        super().__init__()
        self.model = "gpt-4-turbo-2024-04-09"
        self.client = AsyncOpenAI(api_key=os.getenv('OPENAI_API_KEY'), base_url='http://31.172.78.152:9000/v1')
        self.history = UserHistoryManager()
        self.max_retries = 5
        self.max_tokens = 125096
        self.config_tokens = 1024
        self.max_history_size = 10
        self.n_choices = n_choices
        self.retry_delay = 5
        self.max_retries = 3
        self.retries = 0
        self.show_tokens = False
        self.args = {
            "temperature": 0.13, "max_tokens": 4095, "top_p": 1, "frequency_penalty": 0, "presence_penalty": 0.8,
            "stop": None
        }

    async def add_to_history(self, user_id, role, content):
        await self.history.add_to_history(user_id, role, content)

    async def reset_history(self, user_id, content=''):
        await self.history.reset_history(user_id, content)

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
        while self.retries < self.max_retries:
            try:
                if user_id not in self.history.user_dialogs:
                    await self.reset_history(user_id)

                await self.add_to_history(user_id, role="user", content=query)

                token_count = self._count_tokens(self.history.user_dialogs[user_id])
                exceeded_max_tokens = token_count + self.config_tokens > self.max_tokens
                exceeded_max_history_size = len(self.history.user_dialogs[user_id]) > self.max_history_size

                if exceeded_max_tokens or exceeded_max_history_size:
                    logger.info(f'Chat history for chat ID {user_id} is too long. Summarising...')
                    try:
                        summary = await self._summarise(self.history.user_dialogs[user_id][:-1])
                        logger.info(f'Summary: {summary}')
                        await self.reset_history(user_id)
                        await self.add_to_history(user_id, role="assistant", content=summary)
                        await self.add_to_history(user_id, role="user", content=query)
                        logger.info("Dialog From summary: %s", self.history.user_dialogs[user_id])
                    except Exception as e:
                        logger.info(f'Error while summarising chat history: {str(e)}. Popping elements instead...')
                        await self.history.trim_history(user_id, self.max_history_size)
                        logger.info("Dialog From summary exception: %s", self.history.user_dialogs[user_id])

                return await self.client.chat.completions.create(model=self.model,
                                                                 messages=self.history.user_dialogs[user_id],
                                                                 **self.args)

            except Exception as err:
                self.retries += 1
                logger.info("Dialog From custom exception: %s", self.history.user_dialogs[user_id])
                if self.retries == self.max_retries:
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
            encoding = tiktoken.get_encoding("gpt-4-turbo-preview")

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
