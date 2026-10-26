from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.segment_conflict_error import SegmentConflictError
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.segment_correction import SegmentCorrection





T = TypeVar("T", bound="SegmentConflict")



@_attrs_define
class SegmentConflict:
    """
        Attributes:
            error (SegmentConflictError):
            current (SegmentCorrection | Unset):
     """

    error: SegmentConflictError
    current: SegmentCorrection | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.segment_correction import SegmentCorrection # noqa: PLC0415
        error = self.error.value

        current: dict[str, Any] | Unset = UNSET
        if not isinstance(self.current, Unset):
            current = self.current.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "error": error,
        })
        if current is not UNSET:
            field_dict["current"] = current

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.segment_correction import SegmentCorrection # noqa: PLC0415
        d = dict(src_dict)
        error = SegmentConflictError(d.pop("error"))




        _current = d.pop("current", UNSET)
        current: SegmentCorrection | Unset
        if isinstance(_current,  Unset):
            current = UNSET
        else:
            current = SegmentCorrection.from_dict(_current)




        segment_conflict = cls(
            error=error,
            current=current,
        )


        segment_conflict.additional_properties = d
        return segment_conflict

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
