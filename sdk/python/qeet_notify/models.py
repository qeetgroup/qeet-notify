"""Pydantic models for the Qeet Notify API."""

from __future__ import annotations

from datetime import datetime
from typing import Any, Literal

from pydantic import BaseModel, EmailStr, Field


class TriggerEvent(BaseModel):
    name: str
    subscriber_id: str
    payload: dict[str, Any] = Field(default_factory=dict)


class Subscriber(BaseModel):
    id: str | None = None
    external_id: str
    email: EmailStr | None = None
    phone: str | None = None
    first_name: str | None = None
    last_name: str | None = None
    locale: str | None = None
    timezone: str | None = None
    data: dict[str, Any] = Field(default_factory=dict)
    created_at: datetime | None = None
    updated_at: datetime | None = None


class Template(BaseModel):
    id: str | None = None
    name: str
    channel: Literal["email", "sms", "whatsapp", "inapp", "webhook", "push"]
    subject: str | None = None
    body: str
    status: Literal["draft", "published"] = "draft"
    created_at: datetime | None = None


class Workflow(BaseModel):
    id: str | None = None
    name: str
    status: Literal["active", "paused", "draft"] = "draft"
    trigger_event: str
    created_at: datetime | None = None


class Notification(BaseModel):
    id: str
    subscriber_id: str
    channel: str
    status: str
    subject: str | None = None
    created_at: datetime
