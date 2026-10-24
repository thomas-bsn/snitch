from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.search_result_match_type import SearchResultMatchType
from ..models.search_result_status import SearchResultStatus
from ..models.search_result_visibility import SearchResultVisibility
from typing import cast
import datetime






T = TypeVar("T", bound="SearchResult")



@_attrs_define
class SearchResult:
    """
        Attributes:
            clip_id (int):
            clip_name (str):
            visibility (SearchResultVisibility):
            recorded_at (datetime.datetime):
            status (SearchResultStatus):
            sources (list[str]):
            match_type (SearchResultMatchType):
            context (str):
            matched_text (str):
            timestamp (float):
     """

    clip_id: int
    clip_name: str
    visibility: SearchResultVisibility
    recorded_at: datetime.datetime
    status: SearchResultStatus
    sources: list[str]
    match_type: SearchResultMatchType
    context: str
    matched_text: str
    timestamp: float
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        clip_id = self.clip_id

        clip_name = self.clip_name

        visibility = self.visibility.value

        recorded_at = self.recorded_at.isoformat()

        status = self.status.value

        sources = self.sources



        match_type = self.match_type.value

        context = self.context

        matched_text = self.matched_text

        timestamp = self.timestamp


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "clip_id": clip_id,
            "clip_name": clip_name,
            "visibility": visibility,
            "recorded_at": recorded_at,
            "status": status,
            "sources": sources,
            "match_type": match_type,
            "context": context,
            "matched_text": matched_text,
            "timestamp": timestamp,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clip_id = d.pop("clip_id")

        clip_name = d.pop("clip_name")

        visibility = SearchResultVisibility(d.pop("visibility"))




        recorded_at = datetime.datetime.fromisoformat(d.pop("recorded_at"))




        status = SearchResultStatus(d.pop("status"))




        sources = cast(list[str], d.pop("sources"))


        match_type = SearchResultMatchType(d.pop("match_type"))




        context = d.pop("context")

        matched_text = d.pop("matched_text")

        timestamp = d.pop("timestamp")

        search_result = cls(
            clip_id=clip_id,
            clip_name=clip_name,
            visibility=visibility,
            recorded_at=recorded_at,
            status=status,
            sources=sources,
            match_type=match_type,
            context=context,
            matched_text=matched_text,
            timestamp=timestamp,
        )


        search_result.additional_properties = d
        return search_result

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
