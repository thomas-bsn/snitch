from enum import StrEnum

class WorkerEngineStatusCauseCode(StrEnum):
    CANCELLED = "cancelled"
    DISK_FULL = "disk_full"
    HF_NETWORK = "hf_network"
    INVALID_ORDER = "invalid_order"
    RUNTIME_INCOMPATIBLE = "runtime_incompatible"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
