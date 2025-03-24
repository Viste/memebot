FROM python:3.12-slim-bookworm

WORKDIR /app

COPY . /app

RUN apt-get update && apt-get install -y \
    libgl1-mesa-glx \
    libglib2.0-0

RUN pip install --no-cache-dir -r requirements.txt

CMD ["python", "main.py"]
