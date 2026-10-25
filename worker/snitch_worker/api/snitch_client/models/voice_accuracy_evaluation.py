from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="VoiceAccuracyEvaluation")



@_attrs_define
class VoiceAccuracyEvaluation:
    """
        Attributes:
            reference_samples (int): Échantillons validés servant de référence pour cette mesure.
            test_samples (int): Échantillons validés servant de test : leur clip ne contient aucune référence.
            excluded_samples (int): Échantillons écartés de la mesure car leur clip contient aussi une référence.
            evaluated_automatic (int): Tests qui auraient été attribués automatiquement.
            confirmed (int): Confirmations : attributions automatiques simulées qui désignent la Personne validée par un
                humain.
            precision (float | None): confirmed / evaluated_automatic ; null si aucun test n'est automatique.
            coverage (float | None): evaluated_automatic / test_samples ; null sans test.
     """

    reference_samples: int
    test_samples: int
    excluded_samples: int
    evaluated_automatic: int
    confirmed: int
    precision: float | None
    coverage: float | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        reference_samples = self.reference_samples

        test_samples = self.test_samples

        excluded_samples = self.excluded_samples

        evaluated_automatic = self.evaluated_automatic

        confirmed = self.confirmed

        precision: float | None
        precision = self.precision

        coverage: float | None
        coverage = self.coverage


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "reference_samples": reference_samples,
            "test_samples": test_samples,
            "excluded_samples": excluded_samples,
            "evaluated_automatic": evaluated_automatic,
            "confirmed": confirmed,
            "precision": precision,
            "coverage": coverage,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        reference_samples = d.pop("reference_samples")

        test_samples = d.pop("test_samples")

        excluded_samples = d.pop("excluded_samples")

        evaluated_automatic = d.pop("evaluated_automatic")

        confirmed = d.pop("confirmed")

        def _parse_precision(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        precision = _parse_precision(d.pop("precision"))


        def _parse_coverage(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        coverage = _parse_coverage(d.pop("coverage"))


        voice_accuracy_evaluation = cls(
            reference_samples=reference_samples,
            test_samples=test_samples,
            excluded_samples=excluded_samples,
            evaluated_automatic=evaluated_automatic,
            confirmed=confirmed,
            precision=precision,
            coverage=coverage,
        )


        voice_accuracy_evaluation.additional_properties = d
        return voice_accuracy_evaluation

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
