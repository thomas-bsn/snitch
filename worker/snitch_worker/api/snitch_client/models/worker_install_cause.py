from enum import StrEnum

class WorkerInstallCause(StrEnum):
    DOCKER_UNAVAILABLE = "docker_unavailable"
    GPU_UNAVAILABLE = "gpu_unavailable"
    HF_ACCESS_DENIED = "hf_access_denied"
    HF_CACHE_NOT_WRITABLE = "hf_cache_not_writable"
    IMAGE_PULL_FAILED = "image_pull_failed"
    NETWORK_UNREACHABLE = "network_unreachable"
    REGISTRATION_REJECTED = "registration_rejected"
    SERVICE_START_FAILED = "service_start_failed"
    UNKNOWN = "unknown"
    UNSUPPORTED_PLATFORM = "unsupported_platform"
    VOLUME_NOT_WRITABLE = "volume_not_writable"

    def __str__(self) -> str:
        return str(self.value)
