from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_engine_order_result import VoiceEngineOrderResult





T = TypeVar("T", bound="VoiceEngineOrderResults")



@_attrs_define
class VoiceEngineOrderResults:
    """
        Attributes:
            results (list[VoiceEngineOrderResult]):
     """

    results: list[VoiceEngineOrderResult]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_engine_order_result import VoiceEngineOrderResult # noqa: PLC0415
        results = []
        for results_item_data in self.results:
            results_item = results_item_data.to_dict()
            results.append(results_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "results": results,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_engine_order_result import VoiceEngineOrderResult # noqa: PLC0415
        d = dict(src_dict)
        results = []
        _results = d.pop("results")
        for results_item_data in (_results):
            results_item = VoiceEngineOrderResult.from_dict(results_item_data)



            results.append(results_item)


        voice_engine_order_results = cls(
            results=results,
        )


        voice_engine_order_results.additional_properties = d
        return voice_engine_order_results

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
