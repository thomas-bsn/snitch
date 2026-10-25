from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.attribution import Attribution





T = TypeVar("T", bound="AttributionConflict")



@_attrs_define
class AttributionConflict:
    """
        Attributes:
            error (str):
            attribution (Attribution | Unset):
     """

    error: str
    attribution: Attribution | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.attribution import Attribution # noqa: PLC0415
        error = self.error

        attribution: dict[str, Any] | Unset = UNSET
        if not isinstance(self.attribution, Unset):
            attribution = self.attribution.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "error": error,
        })
        if attribution is not UNSET:
            field_dict["attribution"] = attribution

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.attribution import Attribution # noqa: PLC0415
        d = dict(src_dict)
        error = d.pop("error")

        _attribution = d.pop("attribution", UNSET)
        attribution: Attribution | Unset
        if isinstance(_attribution,  Unset):
            attribution = UNSET
        else:
            attribution = Attribution.from_dict(_attribution)




        attribution_conflict = cls(
            error=error,
            attribution=attribution,
        )


        attribution_conflict.additional_properties = d
        return attribution_conflict

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
