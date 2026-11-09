from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.benchmark_run_input_stage import BenchmarkRunInputStage
from typing import cast






T = TypeVar("T", bound="BenchmarkRunInput")



@_attrs_define
class BenchmarkRunInput:
    """
        Attributes:
            dataset_id (int):
            stage (BenchmarkRunInputStage):
            model_ids (list[str]):
     """

    dataset_id: int
    stage: BenchmarkRunInputStage
    model_ids: list[str]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        dataset_id = self.dataset_id

        stage = self.stage.value

        model_ids = self.model_ids




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "dataset_id": dataset_id,
            "stage": stage,
            "model_ids": model_ids,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        dataset_id = d.pop("dataset_id")

        stage = BenchmarkRunInputStage(d.pop("stage"))




        model_ids = cast(list[str], d.pop("model_ids"))


        benchmark_run_input = cls(
            dataset_id=dataset_id,
            stage=stage,
            model_ids=model_ids,
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
