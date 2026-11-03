from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.explorer_index_result_state import ExplorerIndexResultState
from ..types import UNSET, Unset






T = TypeVar("T", bound="ExplorerIndexResult")



@_attrs_define
class ExplorerIndexResult:
    """
        Attributes:
            path (str):
            state (ExplorerIndexResultState):
            clip_id (int | Unset):
     """

    path: str
    state: ExplorerIndexResultState
    clip_id: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        path = self.path

        state = self.state.value

        clip_id = self.clip_id


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "path": path,
            "state": state,
        })
        if clip_id is not UNSET:
            field_dict["clip_id"] = clip_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        path = d.pop("path")

        state = ExplorerIndexResultState(d.pop("state"))




        clip_id = d.pop("clip_id", UNSET)

        explorer_index_result = cls(
            path=path,
            state=state,
            clip_id=clip_id,
        )


        explorer_index_result.additional_properties = d
        return explorer_index_result

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
