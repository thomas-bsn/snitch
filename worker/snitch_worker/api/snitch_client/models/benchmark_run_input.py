from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.benchmark_run_input_stage import BenchmarkRunInputStage
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.benchmark_run_input_chains_item import BenchmarkRunInputChainsItem





T = TypeVar("T", bound="BenchmarkRunInput")



@_attrs_define
class BenchmarkRunInput:
    """
        Attributes:
            dataset_id (int):
            stage (BenchmarkRunInputStage):
            model_ids (list[str] | Unset): Modèles comparés (empreinte, transcription, diarisation). Absent pour le bout en
                bout.
            chains (list[BenchmarkRunInputChainsItem] | Unset): Bout en bout : paires modèle de diarisation + modèle de
                transcription.
     """

    dataset_id: int
    stage: BenchmarkRunInputStage
    model_ids: list[str] | Unset = UNSET
    chains: list[BenchmarkRunInputChainsItem] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.benchmark_run_input_chains_item import BenchmarkRunInputChainsItem # noqa: PLC0415
        dataset_id = self.dataset_id

        stage = self.stage.value

        model_ids: list[str] | Unset = UNSET
        if not isinstance(self.model_ids, Unset):
            model_ids = self.model_ids



        chains: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.chains, Unset):
            chains = []
            for chains_item_data in self.chains:
                chains_item = chains_item_data.to_dict()
                chains.append(chains_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "dataset_id": dataset_id,
            "stage": stage,
        })
        if model_ids is not UNSET:
            field_dict["model_ids"] = model_ids
        if chains is not UNSET:
            field_dict["chains"] = chains

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.benchmark_run_input_chains_item import BenchmarkRunInputChainsItem # noqa: PLC0415
        d = dict(src_dict)
        dataset_id = d.pop("dataset_id")

        stage = BenchmarkRunInputStage(d.pop("stage"))




        model_ids = cast(list[str], d.pop("model_ids", UNSET))


        _chains = d.pop("chains", UNSET)
        chains: list[BenchmarkRunInputChainsItem] | Unset = UNSET
        if _chains is not UNSET:
            chains = []
            for chains_item_data in _chains:
                chains_item = BenchmarkRunInputChainsItem.from_dict(chains_item_data)



                chains.append(chains_item)


        benchmark_run_input = cls(
            dataset_id=dataset_id,
            stage=stage,
            model_ids=model_ids,
            chains=chains,
        )


        benchmark_run_input.additional_properties = d
        return benchmark_run_input

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
