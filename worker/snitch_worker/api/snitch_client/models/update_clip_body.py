from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.update_clip_body_visibility import UpdateClipBodyVisibility






T = TypeVar("T", bound="UpdateClipBody")



@_attrs_define
class UpdateClipBody:
    """
        Attributes:
            visibility (UpdateClipBodyVisibility): Visibilité du clip : shared = visible par tous les membres du workspace,
                private = visible seulement par le propriétaire
     """

    visibility: UpdateClipBodyVisibility
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        visibility = self.visibility.value


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "visibility": visibility,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        visibility = UpdateClipBodyVisibility(d.pop("visibility"))




        update_clip_body = cls(
            visibility=visibility,
        )


        update_clip_body.additional_properties = d
        return update_clip_body

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
