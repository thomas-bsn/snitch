from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.person import Person





T = TypeVar("T", bound="VoiceGroupNaming")



@_attrs_define
class VoiceGroupNaming:
    """
        Attributes:
            person (Person):
            validated (list[int]):
            skipped (list[int]): Voix déjà validées par quelqu'un d'autre pendant l'opération.
     """

    person: Person
    validated: list[int]
    skipped: list[int]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.person import Person # noqa: PLC0415
        person = self.person.to_dict()

        validated = self.validated



        skipped = self.skipped




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "person": person,
            "validated": validated,
            "skipped": skipped,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.person import Person # noqa: PLC0415
        d = dict(src_dict)
        person = Person.from_dict(d.pop("person"))




        validated = cast(list[int], d.pop("validated"))


        skipped = cast(list[int], d.pop("skipped"))


        voice_group_naming = cls(
            person=person,
            validated=validated,
            skipped=skipped,
        )


        voice_group_naming.additional_properties = d
        return voice_group_naming

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
