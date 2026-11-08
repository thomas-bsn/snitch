from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_engine_wheel import VoiceEngineWheel





T = TypeVar("T", bound="VoiceEngineManifest")



@_attrs_define
class VoiceEngineManifest:
    """
        Attributes:
            engine (str):
            version (str):
            python (str):
            platform (str):
            cuda (str):
            license_ (str):
            kind (str):
            architecture (str):
            schema (int):
            adapter_version (int):
            disk_bytes (int):
            vram_bytes (int):
            wheels (list[VoiceEngineWheel]):
     """

    engine: str
    version: str
    python: str
    platform: str
    cuda: str
    license_: str
    kind: str
    architecture: str
    schema: int
    adapter_version: int
    disk_bytes: int
    vram_bytes: int
    wheels: list[VoiceEngineWheel]





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_engine_wheel import VoiceEngineWheel # noqa: PLC0415
        engine = self.engine

        version = self.version

        python = self.python

        platform = self.platform

        cuda = self.cuda

        license_ = self.license_

        kind = self.kind

        architecture = self.architecture

        schema = self.schema

        adapter_version = self.adapter_version

        disk_bytes = self.disk_bytes

        vram_bytes = self.vram_bytes

        wheels = []
        for wheels_item_data in self.wheels:
            wheels_item = wheels_item_data.to_dict()
            wheels.append(wheels_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "engine": engine,
            "version": version,
            "python": python,
            "platform": platform,
            "cuda": cuda,
            "license": license_,
            "kind": kind,
            "architecture": architecture,
            "schema": schema,
            "adapter_version": adapter_version,
            "disk_bytes": disk_bytes,
            "vram_bytes": vram_bytes,
            "wheels": wheels,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_engine_wheel import VoiceEngineWheel # noqa: PLC0415
        d = dict(src_dict)
        engine = d.pop("engine")

        version = d.pop("version")

        python = d.pop("python")

        platform = d.pop("platform")

        cuda = d.pop("cuda")

        license_ = d.pop("license")

        kind = d.pop("kind")

        architecture = d.pop("architecture")

        schema = d.pop("schema")

        adapter_version = d.pop("adapter_version")

        disk_bytes = d.pop("disk_bytes")

        vram_bytes = d.pop("vram_bytes")

        wheels = []
        _wheels = d.pop("wheels")
        for wheels_item_data in (_wheels):
            wheels_item = VoiceEngineWheel.from_dict(wheels_item_data)



            wheels.append(wheels_item)


        voice_engine_manifest = cls(
            engine=engine,
            version=version,
            python=python,
            platform=platform,
            cuda=cuda,
            license_=license_,
            kind=kind,
            architecture=architecture,
            schema=schema,
            adapter_version=adapter_version,
            disk_bytes=disk_bytes,
            vram_bytes=vram_bytes,
            wheels=wheels,
        )

        return voice_engine_manifest
