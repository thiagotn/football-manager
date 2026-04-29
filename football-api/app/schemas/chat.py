import uuid
from datetime import datetime

from pydantic import BaseModel


class ChatMessage(BaseModel):
    role: str  # "user" or "assistant"
    content: str


class ChatRequest(BaseModel):
    messages: list[ChatMessage]


class ChatUserItem(BaseModel):
    model_config = {"from_attributes": True}

    id: uuid.UUID
    name: str
    whatsapp: str
    chat_enabled: bool
    created_at: datetime


class ChatUsersResponse(BaseModel):
    users: list[ChatUserItem]
    total_enabled: int


class ChatAccessUpdate(BaseModel):
    chat_enabled: bool
