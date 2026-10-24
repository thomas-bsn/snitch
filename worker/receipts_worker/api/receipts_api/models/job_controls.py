from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="JobControls")



@_attrs_define
class JobControls:
    """
        Attributes:
            paused (bool):
            max_concurrency (int):
            updated_at (datetime.datetime):
     """

    paused: bool
    max_concurrency: int
    updated_at: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        paused = self.paused

        max_concurrency = self.max_concurrency

        updated_at = self.updated_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "paused": paused,
            "max_concurrency": max_concurrency,
            "updated_at": updated_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        paused = d.pop("paused")

        max_concurrency = d.pop("max_concurrency")

        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))




        job_controls = cls(
            paused=paused,
            max_concurrency=max_concurrency,
            updated_at=updated_at,
        )

        return job_controls
