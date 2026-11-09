from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.benchmark_dataset import BenchmarkDataset





T = TypeVar("T", bound="BenchmarkDatasetList")



@_attrs_define
class BenchmarkDatasetList:
    """
        Attributes:
            datasets (list[BenchmarkDataset]):
     """

    datasets: list[BenchmarkDataset]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.benchmark_dataset import BenchmarkDataset # noqa: PLC0415
        datasets = []
        for datasets_item_data in self.datasets:
            datasets_item = datasets_item_data.to_dict()
            datasets.append(datasets_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "datasets": datasets,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.benchmark_dataset import BenchmarkDataset # noqa: PLC0415
        d = dict(src_dict)
        datasets = []
        _datasets = d.pop("datasets")
        for datasets_item_data in (_datasets):
            datasets_item = BenchmarkDataset.from_dict(datasets_item_data)



            datasets.append(datasets_item)


        benchmark_dataset_list = cls(
            datasets=datasets,
        )


        benchmark_dataset_list.additional_properties = d
        return benchmark_dataset_list

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
