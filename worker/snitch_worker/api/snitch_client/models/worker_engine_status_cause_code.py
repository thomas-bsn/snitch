from enum import StrEnum

class WorkerEngineStatusCauseCode(StrEnum):
    BUILD_FAILED = "build_failed"
    CANCELLED = "cancelled"
    CUDA_INCOMPATIBLE = "cuda_incompatible"
    DISK_FULL = "disk_full"
    HF_NETWORK = "hf_network"
    INSUFFICIENT_RAM = "insufficient_ram"
    INSUFFICIENT_VRAM = "insufficient_vram"
    INVALID_ORDER = "invalid_order"
    PACKAGE_NOT_FOUND = "package_not_found"
    PLATFORM_INCOMPATIBLE = "platform_incompatible"
    RESOLUTION_FAILED = "resolution_failed"
    RUNTIME_INCOMPATIBLE = "runtime_incompatible"
    RUNTIME_IN_USE = "runtime_in_use"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
