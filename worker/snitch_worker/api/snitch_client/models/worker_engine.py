from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_engine_cause_code import WorkerEngineCauseCode
from ..models.worker_engine_order_cause_code import WorkerEngineOrderCauseCode
from ..models.worker_engine_order_intent import WorkerEngineOrderIntent
from ..models.worker_engine_order_status import WorkerEngineOrderStatus
from ..models.worker_engine_state import WorkerEngineState
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.engine_blockers import EngineBlockers
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
            order_intent (WorkerEngineOrderIntent): Intention du dernier ordre sur cette version.
            order_status (WorkerEngineOrderStatus): Résultat du dernier ordre (un retrait refusé laisse le runtime prêt).
            order_cause_code (WorkerEngineOrderCauseCode): Cause d'échec du dernier ordre.
            waiting (EngineBlockers | None): Ce qu'un retrait enregistré attend encore avant d'être livré au Worker ; null
                sinon.
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
    order_intent: WorkerEngineOrderIntent
    order_status: WorkerEngineOrderStatus
    order_cause_code: WorkerEngineOrderCauseCode
    waiting: EngineBlockers | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.engine_blockers import EngineBlockers # noqa: PLC0415
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



        order_intent = self.order_intent.value

        order_status = self.order_status.value

        order_cause_code = self.order_cause_code.value

        waiting: dict[str, Any] | None
        if isinstance(self.waiting, EngineBlockers):
            waiting = self.waiting.to_dict()
        else:
            waiting = self.waiting


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
            "order_intent": order_intent,
            "order_status": order_status,
            "order_cause_code": order_cause_code,
            "waiting": waiting,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.engine_blockers import EngineBlockers # noqa: PLC0415
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


        order_intent = WorkerEngineOrderIntent(d.pop("order_intent"))




        order_status = WorkerEngineOrderStatus(d.pop("order_status"))




        order_cause_code = WorkerEngineOrderCauseCode(d.pop("order_cause_code"))




        def _parse_waiting(data: object) -> EngineBlockers | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                waiting_type_0 = EngineBlockers.from_dict(data)



                return waiting_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(EngineBlockers | None, data)

        waiting = _parse_waiting(d.pop("waiting"))


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
            order_intent=order_intent,
            order_status=order_status,
            order_cause_code=order_cause_code,
            waiting=waiting,
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
