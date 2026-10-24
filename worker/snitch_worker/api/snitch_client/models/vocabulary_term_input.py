from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="VocabularyTermInput")



@_attrs_define
class VocabularyTermInput:
    """
        Attributes:
            term (str):
            variants (list[str]):
            preferred_spelling (str):
            active (bool):
     """

    term: str
    variants: list[str]
    preferred_spelling: str
    active: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        term = self.term

        variants = self.variants



        preferred_spelling = self.preferred_spelling

        active = self.active


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "term": term,
            "variants": variants,
            "preferred_spelling": preferred_spelling,
            "active": active,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        term = d.pop("term")

        variants = cast(list[str], d.pop("variants"))


        preferred_spelling = d.pop("preferred_spelling")

        active = d.pop("active")

        vocabulary_term_input = cls(
            term=term,
            variants=variants,
            preferred_spelling=preferred_spelling,
            active=active,
        )


        vocabulary_term_input.additional_properties = d
        return vocabulary_term_input

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
