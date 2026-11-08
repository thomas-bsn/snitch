from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_engine_manifest_capabilities_item import VoiceEngineManifestCapabilitiesItem
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
            ram_bytes (int | Unset): Mémoire vive minimale (optionnelle ; absente = aucune contrainte).
            capabilities (list[VoiceEngineManifestCapabilitiesItem] | Unset): Capacités déclarées (optionnelles ; absentes =
                celle du type). La séparation n'est pas une capacité approuvée.
            input_contract (str | Unset): Contrat d'entrée déclaré (optionnel).
            output_contract (str | Unset): Contrat de sortie déclaré (optionnel).
            cpu_fallback (bool | Unset): Champ historique conservé pour la compatibilité des manifestes. Une exécution CPU
                exige une release CPU distincte, installée volontairement.
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
    ram_bytes: int | Unset = UNSET
    capabilities: list[VoiceEngineManifestCapabilitiesItem] | Unset = UNSET
    input_contract: str | Unset = UNSET
    output_contract: str | Unset = UNSET
    cpu_fallback: bool | Unset = UNSET





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



        ram_bytes = self.ram_bytes

        capabilities: list[str] | Unset = UNSET
        if not isinstance(self.capabilities, Unset):
            capabilities = []
            for capabilities_item_data in self.capabilities:
                capabilities_item = capabilities_item_data.value
                capabilities.append(capabilities_item)



        input_contract = self.input_contract

        output_contract = self.output_contract

        cpu_fallback = self.cpu_fallback


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
        if ram_bytes is not UNSET:
            field_dict["ram_bytes"] = ram_bytes
        if capabilities is not UNSET:
            field_dict["capabilities"] = capabilities
        if input_contract is not UNSET:
            field_dict["input_contract"] = input_contract
        if output_contract is not UNSET:
            field_dict["output_contract"] = output_contract
        if cpu_fallback is not UNSET:
            field_dict["cpu_fallback"] = cpu_fallback

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


        ram_bytes = d.pop("ram_bytes", UNSET)

        _capabilities = d.pop("capabilities", UNSET)
        capabilities: list[VoiceEngineManifestCapabilitiesItem] | Unset = UNSET
        if _capabilities is not UNSET:
            capabilities = []
            for capabilities_item_data in _capabilities:
                capabilities_item = VoiceEngineManifestCapabilitiesItem(capabilities_item_data)



                capabilities.append(capabilities_item)


        input_contract = d.pop("input_contract", UNSET)

        output_contract = d.pop("output_contract", UNSET)

        cpu_fallback = d.pop("cpu_fallback", UNSET)

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
            ram_bytes=ram_bytes,
            capabilities=capabilities,
            input_contract=input_contract,
            output_contract=output_contract,
            cpu_fallback=cpu_fallback,
        )

        return voice_engine_manifest
