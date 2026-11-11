from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.benchmark_reference_sources_item import BenchmarkReferenceSourcesItem
from ..models.benchmark_run_stage import BenchmarkRunStage
from ..models.benchmark_run_status import BenchmarkRunStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.benchmark_chain import BenchmarkChain
  from ..models.benchmark_progress import BenchmarkProgress
  from ..models.benchmark_result import BenchmarkResult
  from ..models.benchmark_scoring import BenchmarkScoring





T = TypeVar("T", bound="BenchmarkRun")



@_attrs_define
class BenchmarkRun:
    """
        Attributes:
            id (int):
            dataset_id (int):
            dataset_name (str):
            reference_sources (list[BenchmarkReferenceSourcesItem]): D'où viennent les références des clips certifiés :
                production (sortie du pipeline de production corrigée par des humains, qui avantage le modèle de production)
                et/ou import (STM ou RTTM écrit à la main). L'avertissement de biais ne s'applique que si production y figure.
            stage (BenchmarkRunStage):
            status (BenchmarkRunStatus):
            model_ids (list[str]):
            author_name (str):
            created_at (datetime.datetime):
            progress (BenchmarkProgress):
            results (list[BenchmarkResult]):
            chains (list[BenchmarkChain]): Bout en bout : les paires de modèles que représentent les identifiants des
                résultats.
            scoring (BenchmarkScoring | None): Empreinte : mode de score et règle d'éligibilité sous lesquels le run est
                noté (#397). Null pour les autres étapes.
            finished_at (datetime.datetime | None | Unset):
     """

    id: int
    dataset_id: int
    dataset_name: str
    reference_sources: list[BenchmarkReferenceSourcesItem]
    stage: BenchmarkRunStage
    status: BenchmarkRunStatus
    model_ids: list[str]
    author_name: str
    created_at: datetime.datetime
    progress: BenchmarkProgress
    results: list[BenchmarkResult]
    chains: list[BenchmarkChain]
    scoring: BenchmarkScoring | None
    finished_at: datetime.datetime | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.benchmark_chain import BenchmarkChain # noqa: PLC0415
        from ..models.benchmark_progress import BenchmarkProgress # noqa: PLC0415
        from ..models.benchmark_result import BenchmarkResult # noqa: PLC0415
        from ..models.benchmark_scoring import BenchmarkScoring # noqa: PLC0415
        id = self.id

        dataset_id = self.dataset_id

        dataset_name = self.dataset_name

        reference_sources = []
        for componentsschemas_benchmark_reference_sources_item_data in self.reference_sources:
            componentsschemas_benchmark_reference_sources_item = componentsschemas_benchmark_reference_sources_item_data.value
            reference_sources.append(componentsschemas_benchmark_reference_sources_item)



        stage = self.stage.value

        status = self.status.value

        model_ids = self.model_ids



        author_name = self.author_name

        created_at = self.created_at.isoformat()

        progress = self.progress.to_dict()

        results = []
        for results_item_data in self.results:
            results_item = results_item_data.to_dict()
            results.append(results_item)



        chains = []
        for chains_item_data in self.chains:
            chains_item = chains_item_data.to_dict()
            chains.append(chains_item)



        scoring: dict[str, Any] | None
        if isinstance(self.scoring, BenchmarkScoring):
            scoring = self.scoring.to_dict()
        else:
            scoring = self.scoring

        finished_at: None | str | Unset
        if isinstance(self.finished_at, Unset):
            finished_at = UNSET
        elif isinstance(self.finished_at, datetime.datetime):
            finished_at = self.finished_at.isoformat()
        else:
            finished_at = self.finished_at


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "dataset_id": dataset_id,
            "dataset_name": dataset_name,
            "reference_sources": reference_sources,
            "stage": stage,
            "status": status,
            "model_ids": model_ids,
            "author_name": author_name,
            "created_at": created_at,
            "progress": progress,
            "results": results,
            "chains": chains,
            "scoring": scoring,
        })
        if finished_at is not UNSET:
            field_dict["finished_at"] = finished_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.benchmark_chain import BenchmarkChain # noqa: PLC0415
        from ..models.benchmark_progress import BenchmarkProgress # noqa: PLC0415
        from ..models.benchmark_result import BenchmarkResult # noqa: PLC0415
        from ..models.benchmark_scoring import BenchmarkScoring # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        dataset_id = d.pop("dataset_id")

        dataset_name = d.pop("dataset_name")

        reference_sources = []
        _reference_sources = d.pop("reference_sources")
        for componentsschemas_benchmark_reference_sources_item_data in (_reference_sources):
            componentsschemas_benchmark_reference_sources_item = BenchmarkReferenceSourcesItem(componentsschemas_benchmark_reference_sources_item_data)



            reference_sources.append(componentsschemas_benchmark_reference_sources_item)


        stage = BenchmarkRunStage(d.pop("stage"))




        status = BenchmarkRunStatus(d.pop("status"))




        model_ids = cast(list[str], d.pop("model_ids"))


        author_name = d.pop("author_name")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        progress = BenchmarkProgress.from_dict(d.pop("progress"))




        results = []
        _results = d.pop("results")
        for results_item_data in (_results):
            results_item = BenchmarkResult.from_dict(results_item_data)



            results.append(results_item)


        chains = []
        _chains = d.pop("chains")
        for chains_item_data in (_chains):
            chains_item = BenchmarkChain.from_dict(chains_item_data)



            chains.append(chains_item)


        def _parse_scoring(data: object) -> BenchmarkScoring | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                scoring_type_1 = BenchmarkScoring.from_dict(data)



                return scoring_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(BenchmarkScoring | None, data)

        scoring = _parse_scoring(d.pop("scoring"))


        def _parse_finished_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                finished_at_type_0 = datetime.datetime.fromisoformat(data)



                return finished_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        finished_at = _parse_finished_at(d.pop("finished_at", UNSET))


        benchmark_run = cls(
            id=id,
            dataset_id=dataset_id,
            dataset_name=dataset_name,
            reference_sources=reference_sources,
            stage=stage,
            status=status,
            model_ids=model_ids,
            author_name=author_name,
            created_at=created_at,
            progress=progress,
            results=results,
            chains=chains,
            scoring=scoring,
            finished_at=finished_at,
        )


        benchmark_run.additional_properties = d
        return benchmark_run

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
