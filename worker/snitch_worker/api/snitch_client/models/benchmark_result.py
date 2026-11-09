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
  from ..models.benchmark_result_resources_type_0 import BenchmarkResultResourcesType0





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
                précision, rappel… ; transcription : wer, cer…). Jamais comparables en score brut d'un modèle à l'autre.
            elapsed_seconds (float | None | Unset):
            resources (BenchmarkResultResourcesType0 | None | Unset): RAM/VRAM : non remontés par le Worker, toujours null
                pour l'instant.
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
    resources: BenchmarkResultResourcesType0 | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.benchmark_metrics import BenchmarkMetrics # noqa: PLC0415
        from ..models.benchmark_result_resources_type_0 import BenchmarkResultResourcesType0 # noqa: PLC0415
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

        resources: dict[str, Any] | None | Unset
        if isinstance(self.resources, Unset):
            resources = UNSET
        elif isinstance(self.resources, BenchmarkResultResourcesType0):
            resources = self.resources.to_dict()
        else:
            resources = self.resources


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
        from ..models.benchmark_result_resources_type_0 import BenchmarkResultResourcesType0 # noqa: PLC0415
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


        def _parse_resources(data: object) -> BenchmarkResultResourcesType0 | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                resources_type_0 = BenchmarkResultResourcesType0.from_dict(data)



                return resources_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(BenchmarkResultResourcesType0 | None | Unset, data)

        resources = _parse_resources(d.pop("resources", UNSET))


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
