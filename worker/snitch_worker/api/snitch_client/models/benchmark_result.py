from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.benchmark_result_state import BenchmarkResultState
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.benchmark_metrics import BenchmarkMetrics
  from ..models.benchmark_model_resources import BenchmarkModelResources





T = TypeVar("T", bound="BenchmarkResult")



@_attrs_define
class BenchmarkResult:
    """
        Attributes:
            model_id (str):
            model (str):
            model_version (str):
            dimensions (int):
            state (BenchmarkResultState):
            items_total (int):
            items_scored (int):
            metrics (BenchmarkMetrics): Mesures d'un seul modèle ; les champs dépendent de l'étape (empreinte : eer, top1,
                précision, rappel… ; transcription : wer, cer… ; diarisation : der et ses trois composantes). Jamais comparables
                en score brut d'un modèle à l'autre. Bout en bout : cpwer (erreurs de mots avec attribution des locuteurs), wer
                (sans attribution) et der des mêmes sorties.
            elapsed_seconds (float | None | Unset):
            resources (list[BenchmarkModelResources] | Unset): Coût mesuré par le Worker, par modèle du résultat (les deux
                ou trois d'une chaîne de bout en bout), sur ses tâches terminées du run (#403).
     """

    model_id: str
    model: str
    model_version: str
    dimensions: int
    state: BenchmarkResultState
    items_total: int
    items_scored: int
    metrics: BenchmarkMetrics
    elapsed_seconds: float | None | Unset = UNSET
    resources: list[BenchmarkModelResources] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.benchmark_metrics import BenchmarkMetrics # noqa: PLC0415
        from ..models.benchmark_model_resources import BenchmarkModelResources # noqa: PLC0415
        model_id = self.model_id

        model = self.model

        model_version = self.model_version

        dimensions = self.dimensions

        state = self.state.value

        items_total = self.items_total

        items_scored = self.items_scored

        metrics = self.metrics.to_dict()

        elapsed_seconds: float | None | Unset
        if isinstance(self.elapsed_seconds, Unset):
            elapsed_seconds = UNSET
        else:
            elapsed_seconds = self.elapsed_seconds

        resources: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.resources, Unset):
            resources = []
            for resources_item_data in self.resources:
                resources_item = resources_item_data.to_dict()
                resources.append(resources_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "model_id": model_id,
            "model": model,
            "model_version": model_version,
            "dimensions": dimensions,
            "state": state,
            "items_total": items_total,
            "items_scored": items_scored,
            "metrics": metrics,
        })
        if elapsed_seconds is not UNSET:
            field_dict["elapsed_seconds"] = elapsed_seconds
        if resources is not UNSET:
            field_dict["resources"] = resources

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.benchmark_metrics import BenchmarkMetrics # noqa: PLC0415
        from ..models.benchmark_model_resources import BenchmarkModelResources # noqa: PLC0415
        d = dict(src_dict)
        model_id = d.pop("model_id")

        model = d.pop("model")

        model_version = d.pop("model_version")

        dimensions = d.pop("dimensions")

        state = BenchmarkResultState(d.pop("state"))




        items_total = d.pop("items_total")

        items_scored = d.pop("items_scored")

        metrics = BenchmarkMetrics.from_dict(d.pop("metrics"))




        def _parse_elapsed_seconds(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        elapsed_seconds = _parse_elapsed_seconds(d.pop("elapsed_seconds", UNSET))


        _resources = d.pop("resources", UNSET)
        resources: list[BenchmarkModelResources] | Unset = UNSET
        if _resources is not UNSET:
            resources = []
            for resources_item_data in _resources:
                resources_item = BenchmarkModelResources.from_dict(resources_item_data)



                resources.append(resources_item)


        benchmark_result = cls(
            model_id=model_id,
            model=model,
            model_version=model_version,
            dimensions=dimensions,
            state=state,
            items_total=items_total,
            items_scored=items_scored,
            metrics=metrics,
            elapsed_seconds=elapsed_seconds,
            resources=resources,
        )


        benchmark_result.additional_properties = d
        return benchmark_result

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
