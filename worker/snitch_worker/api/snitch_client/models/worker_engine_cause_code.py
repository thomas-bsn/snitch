from enum import StrEnum

class WorkerEngineCauseCode(StrEnum):
    CANCELLED = "cancelled"
    DISK_FULL = "disk_full"
    HF_NETWORK = "hf_network"
    INVALID_ORDER = "invalid_order"
    RUNTIME_INCOMPATIBLE = "runtime_incompatible"
    RUNTIME_MISSING = "runtime_missing"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
