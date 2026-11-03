from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PeopleToProcessTotals")



@_attrs_define
class PeopleToProcessTotals:
    """
        Attributes:
            clips (int):
            to_confirm (int):
            ambiguous (int):
            unknown (int):
            groups (int):
     """

    clips: int
    to_confirm: int
    ambiguous: int
    unknown: int
    groups: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        clips = self.clips

        to_confirm = self.to_confirm

        ambiguous = self.ambiguous

        unknown = self.unknown

        groups = self.groups


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "clips": clips,
            "to_confirm": to_confirm,
            "ambiguous": ambiguous,
            "unknown": unknown,
            "groups": groups,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clips = d.pop("clips")

        to_confirm = d.pop("to_confirm")

        ambiguous = d.pop("ambiguous")

        unknown = d.pop("unknown")

        groups = d.pop("groups")

        people_to_process_totals = cls(
            clips=clips,
            to_confirm=to_confirm,
            ambiguous=ambiguous,
            unknown=unknown,
            groups=groups,
        )


        people_to_process_totals.additional_properties = d
        return people_to_process_totals

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
