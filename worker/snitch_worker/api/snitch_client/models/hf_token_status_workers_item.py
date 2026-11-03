from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.hf_token_status_workers_item_error_code import HFTokenStatusWorkersItemErrorCode
from ..models.hf_token_status_workers_item_source import HFTokenStatusWorkersItemSource
from ..models.hf_token_status_workers_item_state import HFTokenStatusWorkersItemState






T = TypeVar("T", bound="HFTokenStatusWorkersItem")



@_attrs_define
class HFTokenStatusWorkersItem:
    """
        Attributes:
            id (str):
            name (str):
            source (HFTokenStatusWorkersItemSource):
            state (HFTokenStatusWorkersItemState): pending = non synchronisé
            error_code (HFTokenStatusWorkersItemErrorCode):
     """

    id: str
    name: str
    source: HFTokenStatusWorkersItemSource
    state: HFTokenStatusWorkersItemState
    error_code: HFTokenStatusWorkersItemErrorCode





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        source = self.source.value

        state = self.state.value

        error_code = self.error_code.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "source": source,
            "state": state,
            "error_code": error_code,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        source = HFTokenStatusWorkersItemSource(d.pop("source"))




        state = HFTokenStatusWorkersItemState(d.pop("state"))




        error_code = HFTokenStatusWorkersItemErrorCode(d.pop("error_code"))




        hf_token_status_workers_item = cls(
            id=id,
            name=name,
            source=source,
            state=state,
            error_code=error_code,
        )

        return hf_token_status_workers_item
