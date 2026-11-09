from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="BenchmarkClip")



@_attrs_define
class BenchmarkClip:
    """
        Attributes:
            id (int):
            name (str):
            duration_seconds (float | None):
            segments_total (int):
            segments_corrected (int):
            validated_persons (list[str]):
     """

    id: int
    name: str
    duration_seconds: float | None
    segments_total: int
    segments_corrected: int
    validated_persons: list[str]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        duration_seconds: float | None
        duration_seconds = self.duration_seconds

        segments_total = self.segments_total

        segments_corrected = self.segments_corrected

        validated_persons = self.validated_persons




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "name": name,
            "duration_seconds": duration_seconds,
            "segments_total": segments_total,
            "segments_corrected": segments_corrected,
            "validated_persons": validated_persons,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        def _parse_duration_seconds(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        duration_seconds = _parse_duration_seconds(d.pop("duration_seconds"))


        segments_total = d.pop("segments_total")

        segments_corrected = d.pop("segments_corrected")

        validated_persons = cast(list[str], d.pop("validated_persons"))


        benchmark_clip = cls(
            id=id,
            name=name,
            duration_seconds=duration_seconds,
            segments_total=segments_total,
            segments_corrected=segments_corrected,
            validated_persons=validated_persons,
        )


        benchmark_clip.additional_properties = d
        return benchmark_clip

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
