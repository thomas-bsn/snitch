from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

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
            references (int):
            tests (int):
            persons (int):
            clips (int):
     """

    id: int
    name: str
    author_name: str
    created_at: datetime.datetime
    references: int
    tests: int
    persons: int
    clips: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        author_name = self.author_name

        created_at = self.created_at.isoformat()

        references = self.references

        tests = self.tests

        persons = self.persons

        clips = self.clips


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "name": name,
            "author_name": author_name,
            "created_at": created_at,
            "references": references,
            "tests": tests,
            "persons": persons,
            "clips": clips,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        author_name = d.pop("author_name")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        references = d.pop("references")

        tests = d.pop("tests")

        persons = d.pop("persons")

        clips = d.pop("clips")

        benchmark_dataset = cls(
            id=id,
            name=name,
            author_name=author_name,
            created_at=created_at,
            references=references,
            tests=tests,
            persons=persons,
            clips=clips,
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
