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
            test_clip_ids (list[int]):
            reference_clip_ids (list[int]):
            verified_clip_ids (list[int]): Clips de test dont l'administrateur atteste que le texte et les locuteurs ont été
                relus en entier.
     """

    name: str
    test_clip_ids: list[int]
    reference_clip_ids: list[int]
    verified_clip_ids: list[int]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        test_clip_ids = self.test_clip_ids



        reference_clip_ids = self.reference_clip_ids



        verified_clip_ids = self.verified_clip_ids




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "name": name,
            "test_clip_ids": test_clip_ids,
            "reference_clip_ids": reference_clip_ids,
            "verified_clip_ids": verified_clip_ids,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        test_clip_ids = cast(list[int], d.pop("test_clip_ids"))


        reference_clip_ids = cast(list[int], d.pop("reference_clip_ids"))


        verified_clip_ids = cast(list[int], d.pop("verified_clip_ids"))


        benchmark_dataset_input = cls(
            name=name,
            test_clip_ids=test_clip_ids,
            reference_clip_ids=reference_clip_ids,
            verified_clip_ids=verified_clip_ids,
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
