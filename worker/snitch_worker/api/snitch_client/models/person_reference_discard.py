from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.person_reference_discard_reason import PersonReferenceDiscardReason






T = TypeVar("T", bound="PersonReferenceDiscard")



@_attrs_define
class PersonReferenceDiscard:
    """
        Attributes:
            reason (PersonReferenceDiscardReason): Raison de l'écart : celle de l'empreinte du Locuteur local dans le groupe
                actif (too_short, overlap_only, clipped, low_quality), ou no_fingerprint quand la voix n'a aucune empreinte de
                production dans ce groupe.
            count (int):
     """

    reason: PersonReferenceDiscardReason
    count: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        reason = self.reason.value

        count = self.count


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "reason": reason,
            "count": count,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        reason = PersonReferenceDiscardReason(d.pop("reason"))




        count = d.pop("count")

        person_reference_discard = cls(
            reason=reason,
            count=count,
        )


        person_reference_discard.additional_properties = d
        return person_reference_discard

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
