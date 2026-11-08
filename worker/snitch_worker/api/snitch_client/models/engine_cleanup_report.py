from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="EngineCleanupReport")



@_attrs_define
class EngineCleanupReport:
    """
        Attributes:
            order_id (str):
            digest (str):
            removed (bool):
     """

    order_id: str
    digest: str
    removed: bool





    def to_dict(self) -> dict[str, Any]:
        order_id = self.order_id

        digest = self.digest

        removed = self.removed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "order_id": order_id,
            "digest": digest,
            "removed": removed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        order_id = d.pop("order_id")

        digest = d.pop("digest")

        removed = d.pop("removed")

        engine_cleanup_report = cls(
            order_id=order_id,
            digest=digest,
            removed=removed,
        )

        return engine_cleanup_report
