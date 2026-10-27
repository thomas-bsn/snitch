from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WorkerMachine")



@_attrs_define
class WorkerMachine:
    """ Informations non secrètes déclarées par le Worker ; le système est celui du conteneur.

        Attributes:
            name (str | Unset):
            system (str | Unset):
            processor (str | Unset):
            gpu_name (str | Unset):
            gpu_driver (str | Unset):
            worker_version (str | Unset):
            memory_bytes (int | Unset):
            gpu_memory_bytes (int | Unset):
     """

    name: str | Unset = UNSET
    system: str | Unset = UNSET
    processor: str | Unset = UNSET
    gpu_name: str | Unset = UNSET
    gpu_driver: str | Unset = UNSET
    worker_version: str | Unset = UNSET
    memory_bytes: int | Unset = UNSET
    gpu_memory_bytes: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        system = self.system

        processor = self.processor

        gpu_name = self.gpu_name

        gpu_driver = self.gpu_driver

        worker_version = self.worker_version

        memory_bytes = self.memory_bytes

        gpu_memory_bytes = self.gpu_memory_bytes


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if name is not UNSET:
            field_dict["name"] = name
        if system is not UNSET:
            field_dict["system"] = system
        if processor is not UNSET:
            field_dict["processor"] = processor
        if gpu_name is not UNSET:
            field_dict["gpu_name"] = gpu_name
        if gpu_driver is not UNSET:
            field_dict["gpu_driver"] = gpu_driver
        if worker_version is not UNSET:
            field_dict["worker_version"] = worker_version
        if memory_bytes is not UNSET:
            field_dict["memory_bytes"] = memory_bytes
        if gpu_memory_bytes is not UNSET:
            field_dict["gpu_memory_bytes"] = gpu_memory_bytes

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name", UNSET)

        system = d.pop("system", UNSET)

        processor = d.pop("processor", UNSET)

        gpu_name = d.pop("gpu_name", UNSET)

        gpu_driver = d.pop("gpu_driver", UNSET)

        worker_version = d.pop("worker_version", UNSET)

        memory_bytes = d.pop("memory_bytes", UNSET)

        gpu_memory_bytes = d.pop("gpu_memory_bytes", UNSET)

        worker_machine = cls(
            name=name,
            system=system,
            processor=processor,
            gpu_name=gpu_name,
            gpu_driver=gpu_driver,
            worker_version=worker_version,
            memory_bytes=memory_bytes,
            gpu_memory_bytes=gpu_memory_bytes,
        )

        return worker_machine
