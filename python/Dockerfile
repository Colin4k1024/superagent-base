FROM python:3.11-slim AS base

WORKDIR /app

RUN apt-get update && \
    apt-get install -y --no-install-recommends curl && \
    rm -rf /var/lib/apt/lists/*

COPY pyproject.toml .
RUN pip install --no-cache-dir .

COPY src/ src/
COPY configs/ configs/

EXPOSE 8889

CMD ["uvicorn", "superagent.server:app", "--host", "0.0.0.0", "--port", "8889"]
