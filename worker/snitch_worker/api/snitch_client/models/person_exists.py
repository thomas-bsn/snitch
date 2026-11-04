from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.person_exists_error import PersonExistsError
from typing import cast

if TYPE_CHECKING:
  from ..models.person_exists_candidates_item import PersonExistsCandidatesItem





T = TypeVar("T", bound="PersonExists")



@_attrs_define
class PersonExists:
    """
        Attributes:
            error (PersonExistsError):
            candidates (list[PersonExistsCandidatesItem]):
     """

    error: PersonExistsError
    candidates: list[PersonExistsCandidatesItem]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.person_exists_candidates_item import PersonExistsCandidatesItem # noqa: PLC0415
        error = self.error.value

        candidates = []
        for candidates_item_data in self.candidates:
            candidates_item = candidates_item_data.to_dict()
            candidates.append(candidates_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "error": error,
            "candidates": candidates,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.person_exists_candidates_item import PersonExistsCandidatesItem # noqa: PLC0415
        d = dict(src_dict)
        error = PersonExistsError(d.pop("error"))




        candidates = []
        _candidates = d.pop("candidates")
        for candidates_item_data in (_candidates):
            candidates_item = PersonExistsCandidatesItem.from_dict(candidates_item_data)



            candidates.append(candidates_item)


        person_exists = cls(
            error=error,
            candidates=candidates,
        )


        person_exists.additional_properties = d
        return person_exists

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
