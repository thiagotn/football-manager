FROM python:3.12-slim AS base
ENV PYTHONUNBUFFERED=1 PYTHONDONTWRITEBYTECODE=1
WORKDIR /app

FROM base AS dev
RUN pip install poetry==1.8.2 --no-cache-dir
COPY pyproject.toml poetry.lock* ./
RUN poetry config virtualenvs.create false && poetry install --no-root
COPY . .
EXPOSE 8000
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000", "--reload"]

FROM base AS builder
RUN pip install poetry==1.8.2 --no-cache-dir
COPY pyproject.toml poetry.lock* ./
RUN poetry config virtualenvs.create false && \
    poetry export -f requirements.txt --without-hashes --without dev -o requirements.txt

FROM base AS production
COPY --from=builder /app/requirements.txt .
RUN pip install -r requirements.txt --no-cache-dir
COPY app/ ./app/
RUN addgroup --system app && adduser --system --ingroup app appuser
USER appuser
EXPOSE 8000
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "2"]
