from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.engine_order_intent import EngineOrderIntent
from ..models.engine_order_status import EngineOrderStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.engine_runtime import EngineRuntime





T = TypeVar("T", bound="EngineOrder")



@_attrs_define
class EngineOrder:
    """
        Attributes:
            order_id (str):
            worker_id (str):
            digest (str): Identité signée du manifeste approuvé.
            engine (str):
            version (str):
            intent (EngineOrderIntent):
            generation (int): Par Worker et par engine, strictement croissante : un rapport d'une génération plus ancienne
                est ignoré.
            idempotency_key (str):
            status (EngineOrderStatus):
            runtime (EngineRuntime): Runtime figé dans un ordre d'engine : manifeste signé d'une Release
                (VoiceEngineManifest) ou description d'un engine à la demande (OnDemandEngineRuntime). Seuls les champs communs
                sont requis ici ; le Worker valide chaque forme.
            created_at (datetime.datetime):
            cancel_requested (bool | Unset): Annulation demandée : le Worker s'arrête à son prochain point sûr et confirme
                par un rapport failed/cancelled.
     """

    order_id: str
    worker_id: str
    digest: str
    engine: str
    version: str
    intent: EngineOrderIntent
    generation: int
    idempotency_key: str
    status: EngineOrderStatus
    runtime: EngineRuntime
    created_at: datetime.datetime
    cancel_requested: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.engine_runtime import EngineRuntime # noqa: PLC0415
        order_id = self.order_id

        worker_id = self.worker_id

        digest = self.digest

        engine = self.engine

        version = self.version

        intent = self.intent.value

        generation = self.generation

        idempotency_key = self.idempotency_key

        status = self.status.value

        runtime = self.runtime.to_dict()

        created_at = self.created_at.isoformat()

        cancel_requested = self.cancel_requested


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "order_id": order_id,
            "worker_id": worker_id,
            "digest": digest,
            "engine": engine,
            "version": version,
            "intent": intent,
            "generation": generation,
            "idempotency_key": idempotency_key,
            "status": status,
            "runtime": runtime,
            "created_at": created_at,
        })
        if cancel_requested is not UNSET:
            field_dict["cancel_requested"] = cancel_requested

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.engine_runtime import EngineRuntime # noqa: PLC0415
        d = dict(src_dict)
        order_id = d.pop("order_id")

        worker_id = d.pop("worker_id")

        digest = d.pop("digest")

        engine = d.pop("engine")

        version = d.pop("version")

        intent = EngineOrderIntent(d.pop("intent"))




        generation = d.pop("generation")

        idempotency_key = d.pop("idempotency_key")

        status = EngineOrderStatus(d.pop("status"))




        runtime = EngineRuntime.from_dict(d.pop("runtime"))




        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        cancel_requested = d.pop("cancel_requested", UNSET)

        engine_order = cls(
            order_id=order_id,
            worker_id=worker_id,
            digest=digest,
            engine=engine,
            version=version,
            intent=intent,
            generation=generation,
            idempotency_key=idempotency_key,
            status=status,
            runtime=runtime,
            created_at=created_at,
            cancel_requested=cancel_requested,
        )


        engine_order.additional_properties = d
        return engine_order

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
