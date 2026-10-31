from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReprocessClipMetadataResponse200")



@_attrs_define
class ReprocessClipMetadataResponse200:
    """
        Attributes:
            done (bool):
            next_ (int):
            updated (int):
            unavailable (int):
     """

    done: bool
    next_: int
    updated: int
    unavailable: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        done = self.done

        next_ = self.next_

        updated = self.updated

        unavailable = self.unavailable


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "done": done,
            "next": next_,
            "updated": updated,
            "unavailable": unavailable,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        done = d.pop("done")

        next_ = d.pop("next")

        updated = d.pop("updated")

        unavailable = d.pop("unavailable")

        reprocess_clip_metadata_response_200 = cls(
            done=done,
            next_=next_,
            updated=updated,
            unavailable=unavailable,
        )


        reprocess_clip_metadata_response_200.additional_properties = d
        return reprocess_clip_metadata_response_200

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
