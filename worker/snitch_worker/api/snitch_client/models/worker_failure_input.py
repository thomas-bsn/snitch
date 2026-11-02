from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WorkerFailureInput")



@_attrs_define
class WorkerFailureInput:
    """
        Attributes:
            error (str): Code de Cause d'échec (voir FailureCode). Un code inconnu est enregistré comme worker_failed.
            provider (str | Unset): Optionnel. Fournisseur de l'appel en échec.
            model (str | Unset): Optionnel. Modèle de l'appel en échec.
            revision (str | Unset): Optionnel. Révision du modèle.
            http_status (int | Unset): Optionnel. Statut HTTP renvoyé par le fournisseur.
            duration_ms (int | Unset): Optionnel. Durée de l'appel en échec.
     """

    error: str
    provider: str | Unset = UNSET
    model: str | Unset = UNSET
    revision: str | Unset = UNSET
    http_status: int | Unset = UNSET
    duration_ms: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        error = self.error

        provider = self.provider

        model = self.model

        revision = self.revision

        http_status = self.http_status

        duration_ms = self.duration_ms


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "error": error,
        })
        if provider is not UNSET:
            field_dict["provider"] = provider
        if model is not UNSET:
            field_dict["model"] = model
        if revision is not UNSET:
            field_dict["revision"] = revision
        if http_status is not UNSET:
            field_dict["http_status"] = http_status
        if duration_ms is not UNSET:
            field_dict["duration_ms"] = duration_ms

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        error = d.pop("error")

        provider = d.pop("provider", UNSET)

        model = d.pop("model", UNSET)

        revision = d.pop("revision", UNSET)

        http_status = d.pop("http_status", UNSET)

        duration_ms = d.pop("duration_ms", UNSET)

        worker_failure_input = cls(
            error=error,
            provider=provider,
            model=model,
            revision=revision,
            http_status=http_status,
            duration_ms=duration_ms,
        )

        return worker_failure_input
