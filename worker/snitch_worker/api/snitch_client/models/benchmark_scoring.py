from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.benchmark_scoring_score_mode import BenchmarkScoringScoreMode
from typing import cast

if TYPE_CHECKING:
  from ..models.benchmark_eligibility_rule import BenchmarkEligibilityRule





T = TypeVar("T", bound="BenchmarkScoring")



@_attrs_define
class BenchmarkScoring:
    """ Comment un benchmark d'empreinte note : la fonction de score de production sous le mode et la règle d'éligibilité
    actifs au lancement du run. Un run antérieur à leur enregistrement vaut « centroid » (centroïde non pondéré,
    ancien), sans règle.

        Attributes:
            score_mode (BenchmarkScoringScoreMode):
            score_mode_version (int | None):
            reference_sample_cap_seconds (float | None): Plafond du poids d'un échantillon dans une Référence cumulative.
            eligibility_rule (BenchmarkEligibilityRule | None):
     """

    score_mode: BenchmarkScoringScoreMode
    score_mode_version: int | None
    reference_sample_cap_seconds: float | None
    eligibility_rule: BenchmarkEligibilityRule | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.benchmark_eligibility_rule import BenchmarkEligibilityRule # noqa: PLC0415
        score_mode = self.score_mode.value

        score_mode_version: int | None
        score_mode_version = self.score_mode_version

        reference_sample_cap_seconds: float | None
        reference_sample_cap_seconds = self.reference_sample_cap_seconds

        eligibility_rule: dict[str, Any] | None
        if isinstance(self.eligibility_rule, BenchmarkEligibilityRule):
            eligibility_rule = self.eligibility_rule.to_dict()
        else:
            eligibility_rule = self.eligibility_rule


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "score_mode": score_mode,
            "score_mode_version": score_mode_version,
            "reference_sample_cap_seconds": reference_sample_cap_seconds,
            "eligibility_rule": eligibility_rule,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.benchmark_eligibility_rule import BenchmarkEligibilityRule # noqa: PLC0415
        d = dict(src_dict)
        score_mode = BenchmarkScoringScoreMode(d.pop("score_mode"))




        def _parse_score_mode_version(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        score_mode_version = _parse_score_mode_version(d.pop("score_mode_version"))


        def _parse_reference_sample_cap_seconds(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        reference_sample_cap_seconds = _parse_reference_sample_cap_seconds(d.pop("reference_sample_cap_seconds"))


        def _parse_eligibility_rule(data: object) -> BenchmarkEligibilityRule | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                eligibility_rule_type_1 = BenchmarkEligibilityRule.from_dict(data)



                return eligibility_rule_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(BenchmarkEligibilityRule | None, data)

        eligibility_rule = _parse_eligibility_rule(d.pop("eligibility_rule"))


        benchmark_scoring = cls(
            score_mode=score_mode,
            score_mode_version=score_mode_version,
            reference_sample_cap_seconds=reference_sample_cap_seconds,
            eligibility_rule=eligibility_rule,
        )


        benchmark_scoring.additional_properties = d
        return benchmark_scoring

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
