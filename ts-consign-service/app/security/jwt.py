from __future__ import annotations

import base64
import hashlib
import hmac
import json
import time
from dataclasses import dataclass


@dataclass(frozen=True)
class Authentication:
    present: bool
    roles: frozenset[str]


def parse_optional_bearer(authorization: str | None, now: float | None = None) -> Authentication:
    if not authorization or not authorization.startswith("Bearer "):
        return Authentication(False, frozenset())
    token = authorization[7:]
    try:
        header_part, payload_part, signature_part = token.split(".")
        header = json.loads(_decode(header_part))
        payload = json.loads(_decode(payload_part))
        if header.get("alg") != "HS256":
            raise ValueError("unsupported JWT")
        expected = hmac.new(b"secret", f"{header_part}.{payload_part}".encode(), hashlib.sha256).digest()
        signature = base64.urlsafe_b64decode(_pad(signature_part))
        if not hmac.compare_digest(signature, expected):
            raise ValueError("invalid signature")
        if float(payload["exp"]) * 1000 < (now or time.time()) * 1000:
            raise ValueError("expired JWT")
        roles = payload.get("roles")
        if not isinstance(roles, list):
            raise ValueError("invalid roles")
        return Authentication(True, frozenset(str(role) for role in roles))
    except (KeyError, TypeError, ValueError, json.JSONDecodeError) as exc:
        raise ValueError("invalid JWT") from exc


def _decode(value: str) -> str:
    return base64.urlsafe_b64decode(_pad(value)).decode()


def _pad(value: str) -> str:
    return value + "=" * (-len(value) % 4)
