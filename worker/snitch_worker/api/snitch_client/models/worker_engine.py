from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_engine_cause_code import WorkerEngineCauseCode
from ..models.worker_engine_cleanup import WorkerEngineCleanup
from ..models.worker_engine_state import WorkerEngineState
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.worker_engine_history_entry import WorkerEngineHistoryEntry





T = TypeVar("T", bound="WorkerEngine")



@_attrs_define
class WorkerEngine:
    """
        Attributes:
            digest (str):
            engine (str):
            version (str):
            kind (str):
            state (WorkerEngineState):
            cause_code (WorkerEngineCauseCode):
            bytes_received (int):
            bytes_total (int):
            order_id (str):
            generation (int):
            revoked (bool):
            author_name (str):
            updated_at (datetime.datetime):
            history (list[WorkerEngineHistoryEntry]):
            cancel_requested (bool): Annulation demandée, pas encore confirmée par le Worker.
            cleanup (WorkerEngineCleanup): Nettoyage explicite des fichiers incomplets : non demandé, demandé ou confirmé.
     """

    digest: str
    engine: str
    version: str
    kind: str
    state: WorkerEngineState
    cause_code: WorkerEngineCauseCode
    bytes_received: int
    bytes_total: int
    order_id: str
    generation: int
    revoked: bool
    author_name: str
    updated_at: datetime.datetime
    history: list[WorkerEngineHistoryEntry]
    cancel_requested: bool
    cleanup: WorkerEngineCleanup
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_engine_history_entry import WorkerEngineHistoryEntry # noqa: PLC0415
        digest = self.digest

        engine = self.engine

        version = self.version

        kind = self.kind

        state = self.state.value

        cause_code = self.cause_code.value

        bytes_received = self.bytes_received

        bytes_total = self.bytes_total

        order_id = self.order_id

        generation = self.generation

        revoked = self.revoked

        author_name = self.author_name

        updated_at = self.updated_at.isoformat()

        history = []
        for history_item_data in self.history:
            history_item = history_item_data.to_dict()
            history.append(history_item)



        cancel_requested = self.cancel_requested

        cleanup = self.cleanup.value


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "digest": digest,
            "engine": engine,
            "version": version,
            "kind": kind,
            "state": state,
            "cause_code": cause_code,
            "bytes_received": bytes_received,
            "bytes_total": bytes_total,
            "order_id": order_id,
            "generation": generation,
            "revoked": revoked,
            "author_name": author_name,
            "updated_at": updated_at,
            "history": history,
            "cancel_requested": cancel_requested,
            "cleanup": cleanup,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_engine_history_entry import WorkerEngineHistoryEntry # noqa: PLC0415
        d = dict(src_dict)
        digest = d.pop("digest")

        engine = d.pop("engine")

        version = d.pop("version")

        kind = d.pop("kind")

        state = WorkerEngineState(d.pop("state"))




        cause_code = WorkerEngineCauseCode(d.pop("cause_code"))




        bytes_received = d.pop("bytes_received")

        bytes_total = d.pop("bytes_total")

        order_id = d.pop("order_id")

        generation = d.pop("generation")

        revoked = d.pop("revoked")

        author_name = d.pop("author_name")

        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))




        history = []
        _history = d.pop("history")
        for history_item_data in (_history):
            history_item = WorkerEngineHistoryEntry.from_dict(history_item_data)



            history.append(history_item)


        cancel_requested = d.pop("cancel_requested")

        cleanup = WorkerEngineCleanup(d.pop("cleanup"))




        worker_engine = cls(
            digest=digest,
            engine=engine,
            version=version,
            kind=kind,
            state=state,
            cause_code=cause_code,
            bytes_received=bytes_received,
            bytes_total=bytes_total,
            order_id=order_id,
            generation=generation,
            revoked=revoked,
            author_name=author_name,
            updated_at=updated_at,
            history=history,
            cancel_requested=cancel_requested,
            cleanup=cleanup,
        )


        worker_engine.additional_properties = d
        return worker_engine

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
