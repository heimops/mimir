from ..config import Config
from .azure import AzureBackend
from .base import StorageBackend
from .gcs import GCSBackend
from .s3 import S3Backend


def get_backend(config: Config) -> StorageBackend:
    backend = config.storage_backend.lower()
    if backend == "s3":
        if not config.s3:
            raise ValueError("S3 config required (MIMIR_S3__BUCKET, etc.)")
        return S3Backend(config.s3)
    if backend == "azure":
        if not config.azure:
            raise ValueError("Azure config required (MIMIR_AZURE__CONTAINER, etc.)")
        return AzureBackend(config.azure)
    if backend == "gcs":
        if not config.gcs:
            raise ValueError("GCS config required (MIMIR_GCS__BUCKET, etc.)")
        return GCSBackend(config.gcs)
    raise ValueError(f"Unknown storage backend: {backend!r}. Choose s3, azure, or gcs.")
