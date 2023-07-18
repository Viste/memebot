FROM python:3.9-slim-buster

WORKDIR /app

COPY . /app

ARG CONFIG
ENV CONFIG_FILE=$CONFIG

RUN echo $CONFIG_FILE
RUN echo $CONFIG_FILE > /app/core/config.json

RUN pip install --no-cache-dir -r requirements.txt

CMD ["python", "main.py"]