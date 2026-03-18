"""
Testes unitários — app/core/security.py

Cobre: hash de senha, verificação, JWT (access token) e OTP token.
Não requer banco de dados nem HTTP.
"""
from datetime import timedelta

import pytest

from app.core.security import (
    create_access_token,
    create_otp_token,
    decode_access_token,
    decode_otp_token,
    hash_password,
    verify_password,
)


# ── Senha ────────────────────────────────────────────────────────────────────


def test_hash_password_returns_bcrypt_hash():
    hashed = hash_password("senha123")
    assert hashed.startswith("$2b$")


def test_hash_password_two_calls_differ():
    """bcrypt usa salt aleatório — hashes da mesma senha devem ser diferentes."""
    h1 = hash_password("mesma")
    h2 = hash_password("mesma")
    assert h1 != h2


def test_verify_password_correct():
    hashed = hash_password("correta")
    assert verify_password("correta", hashed) is True


def test_verify_password_wrong():
    hashed = hash_password("correta")
    assert verify_password("errada", hashed) is False


def test_verify_password_empty_fails():
    hashed = hash_password("qualquer")
    assert verify_password("", hashed) is False


# ── Access token (JWT) ────────────────────────────────────────────────────────


def test_create_and_decode_access_token():
    player_id = "550e8400-e29b-41d4-a716-446655440000"
    token = create_access_token(player_id)
    result = decode_access_token(token)
    assert result == player_id


def test_decode_invalid_token_returns_none():
    assert decode_access_token("token.invalido.aqui") is None


def test_decode_tampered_token_returns_none():
    token = create_access_token("player-id")
    assert decode_access_token(token + "adulterado") is None


def test_expired_token_returns_none():
    token = create_access_token("player-id", expires_delta=timedelta(seconds=-1))
    assert decode_access_token(token) is None


# ── OTP token ─────────────────────────────────────────────────────────────────


def test_otp_token_roundtrip():
    whatsapp = "11999990000"
    token = create_otp_token(whatsapp)
    result = decode_otp_token(token)
    assert result == whatsapp


def test_otp_token_rejects_regular_access_token():
    """Um JWT de acesso normal não deve ser aceito como OTP token."""
    token = create_access_token("player-id")
    assert decode_otp_token(token) is None


def test_otp_token_rejects_garbage():
    assert decode_otp_token("lixo") is None
