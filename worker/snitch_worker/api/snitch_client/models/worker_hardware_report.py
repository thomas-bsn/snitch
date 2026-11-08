from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WorkerHardwareReport")



@_attrs_define
class WorkerHardwareReport:
    """
        Attributes:
            python (str): Version majeure.mineure de Python du Worker.
            platform (str): Plateforme, par exemple linux-x86_64.
            cuda (str): Version CUDA utilisable ; vide si aucun GPU CUDA n'est utilisable.
            ram_bytes (int):
            vram_bytes (int):
            disk_free_bytes (int):
            cpu_fallback (bool): Configuration du Worker : autorise l'exécution sur le CPU d'une release CUDA dont le
                contrat l'autorise.
     """

    python: str
    platform: str
    cuda: str
    ram_bytes: int
    vram_bytes: int
    disk_free_bytes: int
    cpu_fallback: bool





    def to_dict(self) -> dict[str, Any]:
        python = self.python

        platform = self.platform

        cuda = self.cuda

        ram_bytes = self.ram_bytes

        vram_bytes = self.vram_bytes

        disk_free_bytes = self.disk_free_bytes

        cpu_fallback = self.cpu_fallback


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "python": python,
            "platform": platform,
            "cuda": cuda,
            "ram_bytes": ram_bytes,
            "vram_bytes": vram_bytes,
            "disk_free_bytes": disk_free_bytes,
            "cpu_fallback": cpu_fallback,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        python = d.pop("python")

        platform = d.pop("platform")

        cuda = d.pop("cuda")

        ram_bytes = d.pop("ram_bytes")

        vram_bytes = d.pop("vram_bytes")

        disk_free_bytes = d.pop("disk_free_bytes")

        cpu_fallback = d.pop("cpu_fallback")

        worker_hardware_report = cls(
            python=python,
            platform=platform,
            cuda=cuda,
            ram_bytes=ram_bytes,
            vram_bytes=vram_bytes,
            disk_free_bytes=disk_free_bytes,
            cpu_fallback=cpu_fallback,
        )

        return worker_hardware_report
