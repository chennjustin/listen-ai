# ListenAI FastAPI NLP service
# Env: NLP_PORT (default 8001)
FROM python:3.12-slim-bookworm

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    NLP_PORT=8001

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app.py .

EXPOSE 8001

CMD ["sh", "-c", "exec python -m uvicorn app:app --host 0.0.0.0 --port ${NLP_PORT:-8001}"]
