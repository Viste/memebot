import json
import os
import re
from aiogram import types, F

class JSONObject:
    def __init__(self, dic):
        vars(self).update(dic)


cfg_file = open(os.path.join(os.path.dirname(__file__), 'config.json'), 'r', encoding='utf8')
config = json.loads(cfg_file.read(), object_hook=JSONObject)
SPAM_LINKS_REGEX = re.compile(r"(https?:\/\/)?(t\.me|waxu|binance|xyz)", re.IGNORECASE)
GROUP_ID = 1564920057

def is_spam(message: types.Message):
    if message.text:
        return SPAM_LINKS_REGEX.search(message.text) is not None
    return False