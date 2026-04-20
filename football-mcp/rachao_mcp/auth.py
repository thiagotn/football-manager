import os


def get_token() -> str:
    token = os.getenv("RACHAO_TOKEN")
    if not token:
        raise RuntimeError(
            "RACHAO_TOKEN não definido — configure a variável de ambiente antes de iniciar o MCP"
        )
    return token


def get_api_url() -> str:
    return os.getenv("RACHAO_API_URL", "https://api.rachao.app/api/v1").rstrip("/")
