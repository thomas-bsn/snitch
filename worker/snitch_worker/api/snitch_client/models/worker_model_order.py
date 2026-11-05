from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_model_order_action import WorkerModelOrderAction
from ..models.worker_model_order_kind import WorkerModelOrderKind
from ..models.worker_model_order_status import WorkerModelOrderStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WorkerModelOrder")



@_attrs_define
class WorkerModelOrder:
    """
        Attributes:
            order_id (str):
            worker_id (str):
            kind (WorkerModelOrderKind):
            action (WorkerModelOrderAction):
            model_id (str):
            engine (str):
            repo (str):
            revision (str):
            idempotency_key (str):
            status (WorkerModelOrderStatus):
            created_at (datetime.datetime):
            hf_token (str | Unset): Jeton Hugging Face central, seulement à la première livraison d'un ordre qui télécharge
                un modèle sous accès restreint.
     """

    order_id: str
    worker_id: str
    kind: WorkerModelOrderKind
    action: WorkerModelOrderAction
    model_id: str
    engine: str
    repo: str
    revision: str
    idempotency_key: str
    status: WorkerModelOrderStatus
    created_at: datetime.datetime
    hf_token: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        order_id = self.order_id

        worker_id = self.worker_id

        kind = self.kind.value

        action = self.action.value

        model_id = self.model_id

        engine = self.engine

        repo = self.repo

        revision = self.revision

        idempotency_key = self.idempotency_key

        status = self.status.value

        created_at = self.created_at.isoformat()

        hf_token = self.hf_token


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "order_id": order_id,
            "worker_id": worker_id,
            "kind": kind,
            "action": action,
            "model_id": model_id,
            "engine": engine,
            "repo": repo,
            "revision": revision,
            "idempotency_key": idempotency_key,
            "status": status,
            "created_at": created_at,
        })
        if hf_token is not UNSET:
            field_dict["hf_token"] = hf_token

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        order_id = d.pop("order_id")

        worker_id = d.pop("worker_id")

        kind = WorkerModelOrderKind(d.pop("kind"))




        action = WorkerModelOrderAction(d.pop("action"))




        model_id = d.pop("model_id")

        engine = d.pop("engine")

        repo = d.pop("repo")

        revision = d.pop("revision")

        idempotency_key = d.pop("idempotency_key")

        status = WorkerModelOrderStatus(d.pop("status"))




        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        hf_token = d.pop("hf_token", UNSET)

        worker_model_order = cls(
            order_id=order_id,
            worker_id=worker_id,
            kind=kind,
            action=action,
            model_id=model_id,
            engine=engine,
            repo=repo,
            revision=revision,
            idempotency_key=idempotency_key,
            status=status,
            created_at=created_at,
            hf_token=hf_token,
        )


        worker_model_order.additional_properties = d
        return worker_model_order

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
