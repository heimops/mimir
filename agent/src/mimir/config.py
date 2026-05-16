from __future__ import annotations

import json
from typing import Any, Optional, Tuple, Type

from pydantic import BaseModel
from pydantic.fields import FieldInfo
from pydantic_settings import BaseSettings, EnvSettingsSource, PydanticBaseSettingsSource, SettingsConfigDict


class _CommaListEnvSource(EnvSettingsSource):
    """Env source that accepts comma-separated strings for list[str] fields.

    pydantic-settings v2 calls json.loads() on all complex-typed fields before
    validators run. This subclass falls back to comma-splitting when the value
    is a plain string that is not valid JSON (e.g. MIMIR_NAMESPACES=prod,staging).
    """

    def decode_complex_value(self, field_name: str, field: FieldInfo, value: Any) -> Any:
        try:
            return super().decode_complex_value(field_name, field, value)
        except (json.JSONDecodeError, ValueError):
            if isinstance(value, str):
                return [s.strip() for s in value.split(",") if s.strip()]
            return value


class S3Config(BaseModel):
    bucket: str
    access_key_id: str
    secret_access_key: str
    region: str = "us-east-1"
    endpoint_url: Optional[str] = None
    provider: str = "AWS"


class AzureConfig(BaseModel):
    container: str
    account: str
    key: str


class GCSConfig(BaseModel):
    bucket: str
    service_account_json: Optional[str] = None


class Config(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="MIMIR_",
        env_nested_delimiter="__",
    )

    namespaces: list[str]
    pvcs: Optional[list[str]] = None

    storage_backend: str

    s3: Optional[S3Config] = None
    azure: Optional[AzureConfig] = None
    gcs: Optional[GCSConfig] = None

    schedule: Optional[str] = None

    backup_image: str = "rclone/rclone:latest"
    backup_job_ttl: int = 3600
    job_service_account: str = ""

    @classmethod
    def settings_customise_sources(
        cls,
        settings_cls: Type[BaseSettings],
        init_settings: PydanticBaseSettingsSource,
        env_settings: PydanticBaseSettingsSource,
        dotenv_settings: PydanticBaseSettingsSource,
        file_secret_settings: PydanticBaseSettingsSource,
    ) -> Tuple[PydanticBaseSettingsSource, ...]:
        return (init_settings, _CommaListEnvSource(settings_cls), dotenv_settings, file_secret_settings)
