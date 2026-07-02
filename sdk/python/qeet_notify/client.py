"""Synchronous and async HTTP client for the Qeet Notify API."""

from __future__ import annotations

import uuid
from typing import Any

import httpx

from .models import Notification, Subscriber, TriggerEvent


_DEFAULT_BASE_URL = "https://notify.api.qeet.in"


class QeetNotify:
    """Thread-safe client for the Qeet Notify REST API."""

    def __init__(self, api_key: str, base_url: str = _DEFAULT_BASE_URL) -> None:
        self._client = httpx.Client(
            base_url=base_url,
            headers={
                "X-Qeet-Api-Key": api_key,
                "Content-Type": "application/json",
                "User-Agent": "qeet-notify-python/0.1.0",
            },
            timeout=30,
        )

    # ---- Events ----------------------------------------------------------------

    def trigger(
        self,
        name: str,
        subscriber_id: str,
        payload: dict[str, Any] | None = None,
        idempotency_key: str | None = None,
    ) -> dict[str, Any]:
        """Trigger a notification event. Returns the accepted event ID."""
        resp = self._client.post(
            "/v1/events",
            json=TriggerEvent(
                name=name,
                subscriber_id=subscriber_id,
                payload=payload or {},
            ).model_dump(),
            headers={"Idempotency-Key": idempotency_key or str(uuid.uuid4())},
        )
        resp.raise_for_status()
        return resp.json()

    # ---- Subscribers -----------------------------------------------------------

    def get_subscriber(self, subscriber_id: str) -> Subscriber:
        resp = self._client.get(f"/v1/subscribers/{subscriber_id}")
        resp.raise_for_status()
        return Subscriber.model_validate(resp.json())

    def create_subscriber(self, subscriber: Subscriber) -> Subscriber:
        resp = self._client.post("/v1/subscribers", json=subscriber.model_dump(exclude_none=True))
        resp.raise_for_status()
        return Subscriber.model_validate(resp.json())

    # ---- Notifications ---------------------------------------------------------

    def list_notifications(self, limit: int = 20, offset: int = 0) -> list[Notification]:
        resp = self._client.get("/v1/notifications", params={"limit": limit, "offset": offset})
        resp.raise_for_status()
        return [Notification.model_validate(n) for n in resp.json().get("data", [])]

    # ---- Lifecycle -------------------------------------------------------------

    def close(self) -> None:
        self._client.close()

    def __enter__(self) -> "QeetNotify":
        return self

    def __exit__(self, *_: Any) -> None:
        self.close()
