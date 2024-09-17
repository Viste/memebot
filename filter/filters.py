from aiogram import types
from aiogram.filters import BaseFilter


class IsForwardedFromChannel(BaseFilter):
    async def __call__(self, message: types.Message) -> bool:
        return message.forward_from_chat and message.forward_from_chat.type == 'channel'
