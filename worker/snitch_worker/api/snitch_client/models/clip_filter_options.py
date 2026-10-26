from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.clip_filter_options_owners_item import ClipFilterOptionsOwnersItem
  from ..models.clip_filter_options_sources_item import ClipFilterOptionsSourcesItem





T = TypeVar("T", bound="ClipFilterOptions")



@_attrs_define
class ClipFilterOptions:
    """
        Attributes:
            sources (list[ClipFilterOptionsSourcesItem]):
            owners (list[ClipFilterOptionsOwnersItem]):
     """

    sources: list[ClipFilterOptionsSourcesItem]
    owners: list[ClipFilterOptionsOwnersItem]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.clip_filter_options_owners_item import ClipFilterOptionsOwnersItem # noqa: PLC0415
        from ..models.clip_filter_options_sources_item import ClipFilterOptionsSourcesItem # noqa: PLC0415
        sources = []
        for sources_item_data in self.sources:
            sources_item = sources_item_data.to_dict()
            sources.append(sources_item)



        owners = []
        for owners_item_data in self.owners:
            owners_item = owners_item_data.to_dict()
            owners.append(owners_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "sources": sources,
            "owners": owners,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.clip_filter_options_owners_item import ClipFilterOptionsOwnersItem # noqa: PLC0415
        from ..models.clip_filter_options_sources_item import ClipFilterOptionsSourcesItem # noqa: PLC0415
        d = dict(src_dict)
        sources = []
        _sources = d.pop("sources")
        for sources_item_data in (_sources):
            sources_item = ClipFilterOptionsSourcesItem.from_dict(sources_item_data)



            sources.append(sources_item)


        owners = []
        _owners = d.pop("owners")
        for owners_item_data in (_owners):
            owners_item = ClipFilterOptionsOwnersItem.from_dict(owners_item_data)



            owners.append(owners_item)


        clip_filter_options = cls(
            sources=sources,
            owners=owners,
        )


        clip_filter_options.additional_properties = d
        return clip_filter_options

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
