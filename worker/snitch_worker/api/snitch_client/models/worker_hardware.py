from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="WorkerHardware")



@_attrs_define
class WorkerHardware:
    """
        Attributes:
            python (str):
            platform (str):
            cuda (str):
            ram_bytes (int):
            vram_bytes (int):
            disk_free_bytes (int):
            cpu_fallback (bool):
            reported_at (datetime.datetime):
     """

    python: str
    platform: str
    cuda: str
    ram_bytes: int
    vram_bytes: int
    disk_free_bytes: int
    cpu_fallback: bool
    reported_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        python = self.python

        platform = self.platform

        cuda = self.cuda

        ram_bytes = self.ram_bytes

        vram_bytes = self.vram_bytes

        disk_free_bytes = self.disk_free_bytes

        cpu_fallback = self.cpu_fallback

        reported_at = self.reported_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "python": python,
            "platform": platform,
            "cuda": cuda,
            "ram_bytes": ram_bytes,
            "vram_bytes": vram_bytes,
            "disk_free_bytes": disk_free_bytes,
            "cpu_fallback": cpu_fallback,
            "reported_at": reported_at,
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

        reported_at = datetime.datetime.fromisoformat(d.pop("reported_at"))




        worker_hardware = cls(
            python=python,
            platform=platform,
            cuda=cuda,
            ram_bytes=ram_bytes,
            vram_bytes=vram_bytes,
            disk_free_bytes=disk_free_bytes,
            cpu_fallback=cpu_fallback,
            reported_at=reported_at,
        )


        worker_hardware.additional_properties = d
        return worker_hardware

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
