from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WorkerRegistrationInput")



@_attrs_define
class WorkerRegistrationInput:
    """
        Attributes:
            name (str):
            max_concurrency (int):
     """

    name: str
    max_concurrency: int





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        max_concurrency = self.max_concurrency


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "max_concurrency": max_concurrency,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        max_concurrency = d.pop("max_concurrency")

        worker_registration_input = cls(
            name=name,
            max_concurrency=max_concurrency,
        )

        return worker_registration_input
