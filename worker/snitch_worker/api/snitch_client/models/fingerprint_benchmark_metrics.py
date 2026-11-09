from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="FingerprintBenchmarkMetrics")



@_attrs_define
class FingerprintBenchmarkMetrics:
    """ Mesures d'un seul modèle, jamais comparables en score brut à celles d'un autre. Précision, rappel, faux positifs et
    faux négatifs sont pris au seuil EER du modèle sur la Personne la mieux notée de chaque test : une mauvaise Personne
    acceptée compte à la fois comme faux positif et comme faux négatif.

        Attributes:
            references (int):
            tests (int):
            persons (int):
            genuine_trials (int):
            impostor_trials (int):
            true_positives (int):
            false_positives (int):
            false_negatives (int):
            ignored_dimension (int):
            eer (float | None | Unset):
            eer_threshold (float | None | Unset):
            top1_accuracy (float | None | Unset):
            precision (float | None | Unset):
            recall (float | None | Unset):
            genuine_mean_score (float | None | Unset):
            impostor_mean_score (float | None | Unset):
     """

    references: int
    tests: int
    persons: int
    genuine_trials: int
    impostor_trials: int
    true_positives: int
    false_positives: int
    false_negatives: int
    ignored_dimension: int
    eer: float | None | Unset = UNSET
    eer_threshold: float | None | Unset = UNSET
    top1_accuracy: float | None | Unset = UNSET
    precision: float | None | Unset = UNSET
    recall: float | None | Unset = UNSET
    genuine_mean_score: float | None | Unset = UNSET
    impostor_mean_score: float | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        references = self.references

        tests = self.tests

        persons = self.persons

        genuine_trials = self.genuine_trials

        impostor_trials = self.impostor_trials

        true_positives = self.true_positives

        false_positives = self.false_positives

        false_negatives = self.false_negatives

        ignored_dimension = self.ignored_dimension

        eer: float | None | Unset
        if isinstance(self.eer, Unset):
            eer = UNSET
        else:
            eer = self.eer

        eer_threshold: float | None | Unset
        if isinstance(self.eer_threshold, Unset):
            eer_threshold = UNSET
        else:
            eer_threshold = self.eer_threshold

        top1_accuracy: float | None | Unset
        if isinstance(self.top1_accuracy, Unset):
            top1_accuracy = UNSET
        else:
            top1_accuracy = self.top1_accuracy

        precision: float | None | Unset
        if isinstance(self.precision, Unset):
            precision = UNSET
        else:
            precision = self.precision

        recall: float | None | Unset
        if isinstance(self.recall, Unset):
            recall = UNSET
        else:
            recall = self.recall

        genuine_mean_score: float | None | Unset
        if isinstance(self.genuine_mean_score, Unset):
            genuine_mean_score = UNSET
        else:
            genuine_mean_score = self.genuine_mean_score

        impostor_mean_score: float | None | Unset
        if isinstance(self.impostor_mean_score, Unset):
            impostor_mean_score = UNSET
        else:
            impostor_mean_score = self.impostor_mean_score


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "references": references,
            "tests": tests,
            "persons": persons,
            "genuine_trials": genuine_trials,
            "impostor_trials": impostor_trials,
            "true_positives": true_positives,
            "false_positives": false_positives,
            "false_negatives": false_negatives,
            "ignored_dimension": ignored_dimension,
        })
        if eer is not UNSET:
            field_dict["eer"] = eer
        if eer_threshold is not UNSET:
            field_dict["eer_threshold"] = eer_threshold
        if top1_accuracy is not UNSET:
            field_dict["top1_accuracy"] = top1_accuracy
        if precision is not UNSET:
            field_dict["precision"] = precision
        if recall is not UNSET:
            field_dict["recall"] = recall
        if genuine_mean_score is not UNSET:
            field_dict["genuine_mean_score"] = genuine_mean_score
        if impostor_mean_score is not UNSET:
            field_dict["impostor_mean_score"] = impostor_mean_score

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        references = d.pop("references")

        tests = d.pop("tests")

        persons = d.pop("persons")

        genuine_trials = d.pop("genuine_trials")

        impostor_trials = d.pop("impostor_trials")

        true_positives = d.pop("true_positives")

        false_positives = d.pop("false_positives")

        false_negatives = d.pop("false_negatives")

        ignored_dimension = d.pop("ignored_dimension")

        def _parse_eer(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        eer = _parse_eer(d.pop("eer", UNSET))


        def _parse_eer_threshold(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        eer_threshold = _parse_eer_threshold(d.pop("eer_threshold", UNSET))


        def _parse_top1_accuracy(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        top1_accuracy = _parse_top1_accuracy(d.pop("top1_accuracy", UNSET))


        def _parse_precision(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        precision = _parse_precision(d.pop("precision", UNSET))


        def _parse_recall(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        recall = _parse_recall(d.pop("recall", UNSET))


        def _parse_genuine_mean_score(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        genuine_mean_score = _parse_genuine_mean_score(d.pop("genuine_mean_score", UNSET))


        def _parse_impostor_mean_score(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        impostor_mean_score = _parse_impostor_mean_score(d.pop("impostor_mean_score", UNSET))


        fingerprint_benchmark_metrics = cls(
            references=references,
            tests=tests,
            persons=persons,
            genuine_trials=genuine_trials,
            impostor_trials=impostor_trials,
            true_positives=true_positives,
            false_positives=false_positives,
            false_negatives=false_negatives,
            ignored_dimension=ignored_dimension,
            eer=eer,
            eer_threshold=eer_threshold,
            top1_accuracy=top1_accuracy,
            precision=precision,
            recall=recall,
            genuine_mean_score=genuine_mean_score,
            impostor_mean_score=impostor_mean_score,
        )


        fingerprint_benchmark_metrics.additional_properties = d
        return fingerprint_benchmark_metrics

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
