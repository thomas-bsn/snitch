from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="BenchmarkDatasetInput")



@_attrs_define
class BenchmarkDatasetInput:
    """
        Attributes:
            name (str):
            reference_speaker_ids (list[int]):
            test_speaker_ids (list[int]):
     """

    name: str
    reference_speaker_ids: list[int]
    test_speaker_ids: list[int]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        reference_speaker_ids = self.reference_speaker_ids



        test_speaker_ids = self.test_speaker_ids




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "name": name,
            "reference_speaker_ids": reference_speaker_ids,
            "test_speaker_ids": test_speaker_ids,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        reference_speaker_ids = cast(list[int], d.pop("reference_speaker_ids"))


        test_speaker_ids = cast(list[int], d.pop("test_speaker_ids"))


        benchmark_dataset_input = cls(
            name=name,
            reference_speaker_ids=reference_speaker_ids,
            test_speaker_ids=test_speaker_ids,
        )


        benchmark_dataset_input.additional_properties = d
        return benchmark_dataset_input

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
