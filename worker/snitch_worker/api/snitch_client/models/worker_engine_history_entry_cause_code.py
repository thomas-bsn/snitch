from enum import StrEnum

class WorkerEngineHistoryEntryCauseCode(StrEnum):
    CUDA_INCOMPATIBLE = "cuda_incompatible"
    DISK_FULL = "disk_full"
    HF_NETWORK = "hf_network"
    INSUFFICIENT_RAM = "insufficient_ram"
    INSUFFICIENT_VRAM = "insufficient_vram"
    INVALID_ORDER = "invalid_order"
    PLATFORM_INCOMPATIBLE = "platform_incompatible"
    RUNTIME_INCOMPATIBLE = "runtime_incompatible"
    RUNTIME_MISSING = "runtime_missing"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
