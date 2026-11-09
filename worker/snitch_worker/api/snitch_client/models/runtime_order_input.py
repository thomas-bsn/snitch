from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.runtime_order_input_intent import RuntimeOrderInputIntent
from ..types import UNSET, Unset






T = TypeVar("T", bound="RuntimeOrderInput")



@_attrs_define
class RuntimeOrderInput:
    """
        Attributes:
            worker_id (str):
            model_id (str):
            intent (RuntimeOrderInputIntent):
            idempotency_key (str):
            digest (str | Unset): Identité de la version visée : signée, ou d'un engine à la demande déjà connu. Exclusif
                avec `version`.
            version (str | Unset): Version de la famille d'engine du modèle (ADR 0056) : Snitch en déduit l'identité et
                l'enregistre si elle est nouvelle. Exclusif avec `digest`.
     """

    worker_id: str
    model_id: str
    intent: RuntimeOrderInputIntent
    idempotency_key: str
    digest: str | Unset = UNSET
    version: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        worker_id = self.worker_id

        model_id = self.model_id

        intent = self.intent.value

        idempotency_key = self.idempotency_key

        digest = self.digest

        version = self.version


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "worker_id": worker_id,
            "model_id": model_id,
            "intent": intent,
            "idempotency_key": idempotency_key,
        })
        if digest is not UNSET:
            field_dict["digest"] = digest
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        worker_id = d.pop("worker_id")

        model_id = d.pop("model_id")

        intent = RuntimeOrderInputIntent(d.pop("intent"))




        idempotency_key = d.pop("idempotency_key")

        digest = d.pop("digest", UNSET)

        version = d.pop("version", UNSET)

        runtime_order_input = cls(
            worker_id=worker_id,
            model_id=model_id,
            intent=intent,
            idempotency_key=idempotency_key,
            digest=digest,
            version=version,
        )


        runtime_order_input.additional_properties = d
        return runtime_order_input

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
