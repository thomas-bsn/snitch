from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.person import Person





T = TypeVar("T", bound="PersonCorrectionResult")



@_attrs_define
class PersonCorrectionResult:
    """
        Attributes:
            person (None | Person): Personne qui reste (cible de fusion, Personne séparée) ; null après suppression.
            attributions (int): Attributions rattachées (fusion) ou retirées (séparation, suppression).
            samples (int): Échantillons vocaux déplacés (fusion) ou effacés des références.
            microphone_rules (int): Désignations de piste micro déplacées (fusion) ou supprimées (suppression).
     """

    person: None | Person
    attributions: int
    samples: int
    microphone_rules: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.person import Person # noqa: PLC0415
        person: dict[str, Any] | None
        if isinstance(self.person, Person):
            person = self.person.to_dict()
        else:
            person = self.person

        attributions = self.attributions

        samples = self.samples

        microphone_rules = self.microphone_rules


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "person": person,
            "attributions": attributions,
            "samples": samples,
            "microphone_rules": microphone_rules,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.person import Person # noqa: PLC0415
        d = dict(src_dict)
        def _parse_person(data: object) -> None | Person:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                person_type_1 = Person.from_dict(data)



                return person_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Person, data)

        person = _parse_person(d.pop("person"))


        attributions = d.pop("attributions")

        samples = d.pop("samples")

        microphone_rules = d.pop("microphone_rules")

        person_correction_result = cls(
            person=person,
            attributions=attributions,
            samples=samples,
            microphone_rules=microphone_rules,
        )


        person_correction_result.additional_properties = d
        return person_correction_result

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
