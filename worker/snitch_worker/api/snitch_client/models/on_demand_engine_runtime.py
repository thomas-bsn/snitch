from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.on_demand_engine_runtime_source import OnDemandEngineRuntimeSource
from typing import cast






T = TypeVar("T", bound="OnDemandEngineRuntime")



@_attrs_define
class OnDemandEngineRuntime:
    """ Engine résolu par pip sur le Worker : aucune signature, aucune roue figée. Les contraintes portent les noms du
    manifeste signé.

        Attributes:
            source (OnDemandEngineRuntimeSource):
            family (str):
            engine (str):
            distribution (str): Nom de la distribution pip.
            version (str):
            kind (str):
            kinds (list[str]):
            python (str):
            platform (str):
            cuda (str): Vide : la famille s'exécute avec ou sans GPU.
            disk_bytes (int):
            vram_bytes (int):
            ram_bytes (int):
            estimated_seconds (int):
     """

    source: OnDemandEngineRuntimeSource
    family: str
    engine: str
    distribution: str
    version: str
    kind: str
    kinds: list[str]
    python: str
    platform: str
    cuda: str
    disk_bytes: int
    vram_bytes: int
    ram_bytes: int
    estimated_seconds: int





    def to_dict(self) -> dict[str, Any]:
        source = self.source.value

        family = self.family

        engine = self.engine

        distribution = self.distribution

        version = self.version

        kind = self.kind

        kinds = self.kinds



        python = self.python

        platform = self.platform

        cuda = self.cuda

        disk_bytes = self.disk_bytes

        vram_bytes = self.vram_bytes

        ram_bytes = self.ram_bytes

        estimated_seconds = self.estimated_seconds


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "source": source,
            "family": family,
            "engine": engine,
            "distribution": distribution,
            "version": version,
            "kind": kind,
            "kinds": kinds,
            "python": python,
            "platform": platform,
            "cuda": cuda,
            "disk_bytes": disk_bytes,
            "vram_bytes": vram_bytes,
            "ram_bytes": ram_bytes,
            "estimated_seconds": estimated_seconds,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        source = OnDemandEngineRuntimeSource(d.pop("source"))




        family = d.pop("family")

        engine = d.pop("engine")

        distribution = d.pop("distribution")

        version = d.pop("version")

        kind = d.pop("kind")

        kinds = cast(list[str], d.pop("kinds"))


        python = d.pop("python")

        platform = d.pop("platform")

        cuda = d.pop("cuda")

        disk_bytes = d.pop("disk_bytes")

        vram_bytes = d.pop("vram_bytes")

        ram_bytes = d.pop("ram_bytes")

        estimated_seconds = d.pop("estimated_seconds")

        on_demand_engine_runtime = cls(
            source=source,
            family=family,
            engine=engine,
            distribution=distribution,
            version=version,
            kind=kind,
            kinds=kinds,
            python=python,
            platform=platform,
            cuda=cuda,
            disk_bytes=disk_bytes,
            vram_bytes=vram_bytes,
            ram_bytes=ram_bytes,
            estimated_seconds=estimated_seconds,
        )

        return on_demand_engine_runtime
