# ListenAI Streamlit dashboard
# Env: GATEWAY_URL (default http://localhost:8000 — use http://gateway:8000 in Compose)
#      STREAMLIT_SERVER_PORT (default 8501)
FROM python:3.12-slim-bookworm

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    STREAMLIT_SERVER_HEADLESS=true \
    STREAMLIT_BROWSER_GATHER_USAGE_STATS=false \
    GATEWAY_URL=http://localhost:8000

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app.py .

EXPOSE 8501

CMD ["sh", "-c", "exec streamlit run app.py --server.address=0.0.0.0 --server.port=${STREAMLIT_SERVER_PORT:-8501}"]
