from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="VoiceModelOrderInput")



@_attrs_define
class VoiceModelOrderInput:
    """
        Attributes:
            model_id (str):
            idempotency_key (str):
            replaces_model_id (str | Unset): Modèle prêt du même type que ce Worker remplace : le nouveau est installé à
                côté sans bascule, puis Snitch décide la bascule et ordonne le retrait de l'ancien. Diarisation seulement
                (voice_replacement_kind_unsupported pour l'empreinte).
            runtime_digest (str | Unset): Version approuvée exacte du runtime. Sans valeur : dernière version approuvée
                compatible ; legacy si aucun registre ne couvre cet engine.
     """

    model_id: str
    idempotency_key: str
    replaces_model_id: str | Unset = UNSET
    runtime_digest: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        model_id = self.model_id

        idempotency_key = self.idempotency_key

        replaces_model_id = self.replaces_model_id

        runtime_digest = self.runtime_digest


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "model_id": model_id,
            "idempotency_key": idempotency_key,
        })
        if replaces_model_id is not UNSET:
            field_dict["replaces_model_id"] = replaces_model_id
        if runtime_digest is not UNSET:
            field_dict["runtime_digest"] = runtime_digest

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        model_id = d.pop("model_id")

        idempotency_key = d.pop("idempotency_key")

        replaces_model_id = d.pop("replaces_model_id", UNSET)

        runtime_digest = d.pop("runtime_digest", UNSET)

        voice_model_order_input = cls(
            model_id=model_id,
            idempotency_key=idempotency_key,
            replaces_model_id=replaces_model_id,
            runtime_digest=runtime_digest,
        )

        return voice_model_order_input
