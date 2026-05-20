from abc import ABC, abstractmethod


class StorageBackend(ABC):
    @abstractmethod
    def env_vars(self) -> dict[str, str]:
        """Environment variables for the backup Job container."""

    @abstractmethod
    def remote_path(self, namespace: str, pvc_name: str, timestamp: str) -> str:
        """Destination path for a single PVC backup."""

    def job_image(self) -> str | None:
        """Return a specific container image, or None to use the configured backup_image."""
        return None

    def job_args(self, remote_path: str) -> list[str]:
        """Command args for the backup Job container."""
        return ["copy", "/data", remote_path, "--no-check-dest", "--progress"]
