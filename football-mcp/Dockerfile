FROM python:3.11-slim

WORKDIR /app

COPY pyproject.toml ./
# --prerelease=allow: mcp==2.0.0b1 (pin exato) puxa mcp-types==2.0.0b1, pré-release transitivo
RUN pip install --no-cache-dir uv && uv pip install --system --prerelease=allow -e .

COPY rachao_mcp/ ./rachao_mcp/

ENV PYTHONUNBUFFERED=1
EXPOSE 8080

CMD ["python", "-m", "rachao_mcp"]
