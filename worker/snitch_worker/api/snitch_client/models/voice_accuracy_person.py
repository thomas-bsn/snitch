from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.person_ref import PersonRef





T = TypeVar("T", bound="VoiceAccuracyPerson")



@_attrs_define
class VoiceAccuracyPerson:
    """
        Attributes:
            person (PersonRef):
            reference_samples (int):
            test_samples (int):
            automatic_predictions (int): Tests attribués automatiquement à cette Personne, à raison ou non.
            confirmed (int):
            precision (float | None):
     """

    person: PersonRef
    reference_samples: int
    test_samples: int
    automatic_predictions: int
    confirmed: int
    precision: float | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.person_ref import PersonRef # noqa: PLC0415
        person = self.person.to_dict()

        reference_samples = self.reference_samples

        test_samples = self.test_samples

        automatic_predictions = self.automatic_predictions

        confirmed = self.confirmed

        precision: float | None
        precision = self.precision


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "person": person,
            "reference_samples": reference_samples,
            "test_samples": test_samples,
            "automatic_predictions": automatic_predictions,
            "confirmed": confirmed,
            "precision": precision,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.person_ref import PersonRef # noqa: PLC0415
        d = dict(src_dict)
        person = PersonRef.from_dict(d.pop("person"))




        reference_samples = d.pop("reference_samples")

        test_samples = d.pop("test_samples")

        automatic_predictions = d.pop("automatic_predictions")

        confirmed = d.pop("confirmed")

        def _parse_precision(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        precision = _parse_precision(d.pop("precision"))


        voice_accuracy_person = cls(
            person=person,
            reference_samples=reference_samples,
            test_samples=test_samples,
            automatic_predictions=automatic_predictions,
            confirmed=confirmed,
            precision=precision,
        )


        voice_accuracy_person.additional_properties = d
        return voice_accuracy_person

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
