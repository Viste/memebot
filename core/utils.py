import json
import os


class JSONObject:
    def __init__(self, dic):
        vars(self).update(dic)


cfg_file = open(os.path.join(os.path.dirname(__file__), 'config.json'), 'r', encoding='utf8')
config = json.loads(cfg_file.read(), object_hook=JSONObject)
