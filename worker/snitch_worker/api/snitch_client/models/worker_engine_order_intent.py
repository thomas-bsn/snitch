from enum import StrEnum

class WorkerEngineOrderIntent(StrEnum):
    INSTALL = "install"
    UNINSTALL = "uninstall"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
