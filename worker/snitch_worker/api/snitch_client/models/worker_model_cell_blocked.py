from enum import StrEnum

class WorkerModelCellBlocked(StrEnum):
    ORDER_STALLED = "order_stalled"
    VALUE_0 = ""
    WORKER_OFFLINE = "worker_offline"

    def __str__(self) -> str:
        return str(self.value)
