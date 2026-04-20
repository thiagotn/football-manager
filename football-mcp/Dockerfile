FROM python:3.11-slim

WORKDIR /app

COPY pyproject.toml ./
RUN pip install --no-cache-dir uv && uv pip install --system -e .

COPY rachao_mcp/ ./rachao_mcp/

ENV PYTHONUNBUFFERED=1
EXPOSE 8080

CMD ["python", "-m", "rachao_mcp"]
