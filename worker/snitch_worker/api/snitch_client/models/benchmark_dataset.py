from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.benchmark_dataset_stages_item import BenchmarkDatasetStagesItem
from typing import cast
import datetime






T = TypeVar("T", bound="BenchmarkDataset")



@_attrs_define
class BenchmarkDataset:
    """
        Attributes:
            id (int):
            name (str):
            author_name (str):
            created_at (datetime.datetime):
            stages (list[BenchmarkDatasetStagesItem]): Étapes que le corpus permet de mesurer. Une étape listée n'est pas
                forcément déjà exécutable.
            test_clips (int):
            reference_clips (int):
            verified_clips (int):
            persons (int):
            duration_seconds (float):
     """

    id: int
    name: str
    author_name: str
    created_at: datetime.datetime
    stages: list[BenchmarkDatasetStagesItem]
    test_clips: int
    reference_clips: int
    verified_clips: int
    persons: int
    duration_seconds: float
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        author_name = self.author_name

        created_at = self.created_at.isoformat()

        stages = []
        for stages_item_data in self.stages:
            stages_item = stages_item_data.value
            stages.append(stages_item)



        test_clips = self.test_clips

        reference_clips = self.reference_clips

        verified_clips = self.verified_clips

        persons = self.persons

        duration_seconds = self.duration_seconds


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "name": name,
            "author_name": author_name,
            "created_at": created_at,
            "stages": stages,
            "test_clips": test_clips,
            "reference_clips": reference_clips,
            "verified_clips": verified_clips,
            "persons": persons,
            "duration_seconds": duration_seconds,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        author_name = d.pop("author_name")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        stages = []
        _stages = d.pop("stages")
        for stages_item_data in (_stages):
            stages_item = BenchmarkDatasetStagesItem(stages_item_data)



            stages.append(stages_item)


        test_clips = d.pop("test_clips")

        reference_clips = d.pop("reference_clips")

        verified_clips = d.pop("verified_clips")

        persons = d.pop("persons")

        duration_seconds = d.pop("duration_seconds")

        benchmark_dataset = cls(
            id=id,
            name=name,
            author_name=author_name,
            created_at=created_at,
            stages=stages,
            test_clips=test_clips,
            reference_clips=reference_clips,
            verified_clips=verified_clips,
            persons=persons,
            duration_seconds=duration_seconds,
        )


        benchmark_dataset.additional_properties = d
        return benchmark_dataset

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
