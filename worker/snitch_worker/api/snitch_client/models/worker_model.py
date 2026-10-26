from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WorkerModel")



@_attrs_define
class WorkerModel:
    """
        Attributes:
            provider (str):
            model (str):
            version (str | Unset):
            endpoint (str | Unset): Hôte et port seulement (hote:port), jamais de schéma, de chemin, de paramètre ni
                d’identifiants.
     """

    provider: str
    model: str
    version: str | Unset = UNSET
    endpoint: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        provider = self.provider

        model = self.model

        version = self.version

        endpoint = self.endpoint


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "provider": provider,
            "model": model,
        })
        if version is not UNSET:
            field_dict["version"] = version
        if endpoint is not UNSET:
            field_dict["endpoint"] = endpoint

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        provider = d.pop("provider")

        model = d.pop("model")

        version = d.pop("version", UNSET)

        endpoint = d.pop("endpoint", UNSET)

        worker_model = cls(
            provider=provider,
            model=model,
            version=version,
            endpoint=endpoint,
        )

        return worker_model
