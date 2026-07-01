from __future__ import annotations

import base64
import hashlib
import hmac
import json
import time


def validate_optional_bearer(authorization: str | None, now: float | None = None) -> bool:
    if not authorization or not authorization.startswith("Bearer "):
        return True
    token = authorization[7:]
    try:
        header_part, payload_part, signature_part = token.split(".")
        header = json.loads(_decode(header_part))
        payload = json.loads(_decode(payload_part))
        if header.get("alg") != "HS256":
            return False
        expected = hmac.new(b"secret", f"{header_part}.{payload_part}".encode(), hashlib.sha256).digest()
        signature = base64.urlsafe_b64decode(_pad(signature_part))
        expiration = payload["exp"]
        return hmac.compare_digest(signature, expected) and float(expiration) * 1000 >= (now or time.time()) * 1000
    except (KeyError, TypeError, ValueError, json.JSONDecodeError):
        return False


def _decode(value: str) -> str:
    return base64.urlsafe_b64decode(_pad(value)).decode()


def _pad(value: str) -> str:
    return value + "=" * (-len(value) % 4)
