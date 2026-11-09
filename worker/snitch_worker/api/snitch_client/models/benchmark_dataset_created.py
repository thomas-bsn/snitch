from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.benchmark_dataset_created_stages_item import BenchmarkDatasetCreatedStagesItem
from typing import cast






T = TypeVar("T", bound="BenchmarkDatasetCreated")



@_attrs_define
class BenchmarkDatasetCreated:
    """
        Attributes:
            id (int):
            stages (list[BenchmarkDatasetCreatedStagesItem]):
     """

    id: int
    stages: list[BenchmarkDatasetCreatedStagesItem]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        stages = []
        for stages_item_data in self.stages:
            stages_item = stages_item_data.value
            stages.append(stages_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "stages": stages,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        stages = []
        _stages = d.pop("stages")
        for stages_item_data in (_stages):
            stages_item = BenchmarkDatasetCreatedStagesItem(stages_item_data)



            stages.append(stages_item)


        benchmark_dataset_created = cls(
            id=id,
            stages=stages,
        )


        benchmark_dataset_created.additional_properties = d
        return benchmark_dataset_created

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
