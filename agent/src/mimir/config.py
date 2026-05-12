from __future__ import annotations

from typing import Optional

from pydantic import BaseModel, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class S3Config(BaseModel):
    bucket: str
    access_key_id: str
    secret_access_key: str
    region: str = "us-east-1"
    endpoint_url: Optional[str] = None  # set for IBM COS or custom S3-compatible endpoints
    provider: str = "AWS"  # rclone provider: AWS, IBMCOS, Minio, Other…


class AzureConfig(BaseModel):
    container: str
    account: str
    key: str


class GCSConfig(BaseModel):
    bucket: str
    # JSON string of the service account credentials; if omitted, falls back to ADC
    service_account_json: Optional[str] = None


class Config(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="MIMIR_",
        env_nested_delimiter="__",
    )

    # Comma-separated list of namespaces to back up
    namespaces: list[str]
    # Optional comma-separated list of PVC names; omit to back up all PVCs
    pvcs: Optional[list[str]] = None

    storage_backend: str  # "s3" | "azure" | "gcs"

    s3: Optional[S3Config] = None
    azure: Optional[AzureConfig] = None
    gcs: Optional[GCSConfig] = None

    # Cron expression for scheduled mode (e.g. "0 2 * * *")
    schedule: Optional[str] = None

    # Image used by the rclone backup Jobs
    backup_image: str = "rclone/rclone:latest"
    # TTL (seconds) before Kubernetes auto-deletes completed/failed Jobs
    backup_job_ttl: int = 3600
    # ServiceAccount for backup Jobs (must exist in each target namespace)
    job_service_account: str = ""

    @field_validator("namespaces", mode="before")
    @classmethod
    def _split_namespaces(cls, v):
        if isinstance(v, str):
            return [ns.strip() for ns in v.split(",") if ns.strip()]
        return v

    @field_validator("pvcs", mode="before")
    @classmethod
    def _split_pvcs(cls, v):
        if isinstance(v, str) and v:
            return [p.strip() for p in v.split(",") if p.strip()]
        return v
