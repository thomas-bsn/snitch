from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.engine_runtime_source import EngineRuntimeSource
from ..types import UNSET, Unset






T = TypeVar("T", bound="EngineRuntime")



@_attrs_define
class EngineRuntime:
    """ Runtime figé dans un ordre d'engine : manifeste signé d'une Release (VoiceEngineManifest) ou description d'un engine
    à la demande (OnDemandEngineRuntime). Seuls les champs communs sont requis ici ; le Worker valide chaque forme.

        Attributes:
            engine (str):
            version (str):
            kind (str):
            python (str):
            platform (str):
            cuda (str):
            disk_bytes (int):
            vram_bytes (int):
            source (EngineRuntimeSource | Unset): Présent seulement pour un engine à la demande.
     """

    engine: str
    version: str
    kind: str
    python: str
    platform: str
    cuda: str
    disk_bytes: int
    vram_bytes: int
    source: EngineRuntimeSource | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        engine = self.engine

        version = self.version

        kind = self.kind

        python = self.python

        platform = self.platform

        cuda = self.cuda

        disk_bytes = self.disk_bytes

        vram_bytes = self.vram_bytes

        source: str | Unset = UNSET
        if not isinstance(self.source, Unset):
            source = self.source.value



        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "engine": engine,
            "version": version,
            "kind": kind,
            "python": python,
            "platform": platform,
            "cuda": cuda,
            "disk_bytes": disk_bytes,
            "vram_bytes": vram_bytes,
        })
        if source is not UNSET:
            field_dict["source"] = source

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        engine = d.pop("engine")

        version = d.pop("version")

        kind = d.pop("kind")

        python = d.pop("python")

        platform = d.pop("platform")

        cuda = d.pop("cuda")

        disk_bytes = d.pop("disk_bytes")

        vram_bytes = d.pop("vram_bytes")

        _source = d.pop("source", UNSET)
        source: EngineRuntimeSource | Unset
        if isinstance(_source,  Unset):
            source = UNSET
        else:
            source = EngineRuntimeSource(_source)




        engine_runtime = cls(
            engine=engine,
            version=version,
            kind=kind,
            python=python,
            platform=platform,
            cuda=cuda,
            disk_bytes=disk_bytes,
            vram_bytes=vram_bytes,
            source=source,
        )


        engine_runtime.additional_properties = d
        return engine_runtime

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
