from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="JobControlsInput")



@_attrs_define
class JobControlsInput:
    """
        Attributes:
            paused (bool):
            max_concurrency (int):
     """

    paused: bool
    max_concurrency: int





    def to_dict(self) -> dict[str, Any]:
        paused = self.paused

        max_concurrency = self.max_concurrency


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "paused": paused,
            "max_concurrency": max_concurrency,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        paused = d.pop("paused")

        max_concurrency = d.pop("max_concurrency")

        job_controls_input = cls(
            paused=paused,
            max_concurrency=max_concurrency,
        )

        return job_controls_input
