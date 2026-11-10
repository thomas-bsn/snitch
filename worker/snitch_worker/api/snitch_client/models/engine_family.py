from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="EngineFamily")



@_attrs_define
class EngineFamily:
    """
        Attributes:
            family (str):
            engine (str): Nom d'engine utilisé par le registre, les ordres et l'inventaire.
            default_version (str):
            kinds (list[str]):
            python (str):
            platform (str):
            cuda (str): Vide : tourne avec ou sans GPU.
            vram_bytes (int):
            ram_bytes (int):
            disk_bytes (int):
            estimated_seconds (int):
            recent_versions (list[str]): Dernières versions stables de la bibliothèque publiées sur l'index de paquets, la
                plus récente d'abord (10 au plus). Vide quand l'index est injoignable : la version par défaut reste alors la
                seule proposition.
     """

    family: str
    engine: str
    default_version: str
    kinds: list[str]
    python: str
    platform: str
    cuda: str
    vram_bytes: int
    ram_bytes: int
    disk_bytes: int
    estimated_seconds: int
    recent_versions: list[str]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        family = self.family

        engine = self.engine

        default_version = self.default_version

        kinds = self.kinds



        python = self.python

        platform = self.platform

        cuda = self.cuda

        vram_bytes = self.vram_bytes

        ram_bytes = self.ram_bytes

        disk_bytes = self.disk_bytes

        estimated_seconds = self.estimated_seconds

        recent_versions = self.recent_versions




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "family": family,
            "engine": engine,
            "default_version": default_version,
            "kinds": kinds,
            "python": python,
            "platform": platform,
            "cuda": cuda,
            "vram_bytes": vram_bytes,
            "ram_bytes": ram_bytes,
            "disk_bytes": disk_bytes,
            "estimated_seconds": estimated_seconds,
            "recent_versions": recent_versions,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        family = d.pop("family")

        engine = d.pop("engine")

        default_version = d.pop("default_version")

        kinds = cast(list[str], d.pop("kinds"))


        python = d.pop("python")

        platform = d.pop("platform")

        cuda = d.pop("cuda")

        vram_bytes = d.pop("vram_bytes")

        ram_bytes = d.pop("ram_bytes")

        disk_bytes = d.pop("disk_bytes")

        estimated_seconds = d.pop("estimated_seconds")

        recent_versions = cast(list[str], d.pop("recent_versions"))


        engine_family = cls(
            family=family,
            engine=engine,
            default_version=default_version,
            kinds=kinds,
            python=python,
            platform=platform,
            cuda=cuda,
            vram_bytes=vram_bytes,
            ram_bytes=ram_bytes,
            disk_bytes=disk_bytes,
            estimated_seconds=estimated_seconds,
            recent_versions=recent_versions,
        )


        engine_family.additional_properties = d
        return engine_family

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
