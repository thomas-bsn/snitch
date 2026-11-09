from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.benchmark_clip import BenchmarkClip





T = TypeVar("T", bound="BenchmarkClipList")



@_attrs_define
class BenchmarkClipList:
    """
        Attributes:
            clips (list[BenchmarkClip]):
     """

    clips: list[BenchmarkClip]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.benchmark_clip import BenchmarkClip # noqa: PLC0415
        clips = []
        for clips_item_data in self.clips:
            clips_item = clips_item_data.to_dict()
            clips.append(clips_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "clips": clips,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.benchmark_clip import BenchmarkClip # noqa: PLC0415
        d = dict(src_dict)
        clips = []
        _clips = d.pop("clips")
        for clips_item_data in (_clips):
            clips_item = BenchmarkClip.from_dict(clips_item_data)



            clips.append(clips_item)


        benchmark_clip_list = cls(
            clips=clips,
        )


        benchmark_clip_list.additional_properties = d
        return benchmark_clip_list

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
