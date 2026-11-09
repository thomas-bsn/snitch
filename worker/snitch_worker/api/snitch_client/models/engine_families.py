from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.engine_family import EngineFamily





T = TypeVar("T", bound="EngineFamilies")



@_attrs_define
class EngineFamilies:
    """
        Attributes:
            families (list[EngineFamily]):
     """

    families: list[EngineFamily]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.engine_family import EngineFamily # noqa: PLC0415
        families = []
        for families_item_data in self.families:
            families_item = families_item_data.to_dict()
            families.append(families_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "families": families,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.engine_family import EngineFamily # noqa: PLC0415
        d = dict(src_dict)
        families = []
        _families = d.pop("families")
        for families_item_data in (_families):
            families_item = EngineFamily.from_dict(families_item_data)



            families.append(families_item)


        engine_families = cls(
            families=families,
        )


        engine_families.additional_properties = d
        return engine_families

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
