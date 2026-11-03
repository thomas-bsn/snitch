from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.get_worker_updates_response_200_workers_item_state import GetWorkerUpdatesResponse200WorkersItemState
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="GetWorkerUpdatesResponse200WorkersItem")



@_attrs_define
class GetWorkerUpdatesResponse200WorkersItem:
    """
        Attributes:
            id (str):
            name (str):
            state (GetWorkerUpdatesResponse200WorkersItemState):
            version (str):
            digest (str):
            cause_code (str):
            reported_at (datetime.datetime | None | Unset):
     """

    id: str
    name: str
    state: GetWorkerUpdatesResponse200WorkersItemState
    version: str
    digest: str
    cause_code: str
    reported_at: datetime.datetime | None | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        state = self.state.value

        version = self.version

        digest = self.digest

        cause_code = self.cause_code

        reported_at: None | str | Unset
        if isinstance(self.reported_at, Unset):
            reported_at = UNSET
        elif isinstance(self.reported_at, datetime.datetime):
            reported_at = self.reported_at.isoformat()
        else:
            reported_at = self.reported_at


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "state": state,
            "version": version,
            "digest": digest,
            "cause_code": cause_code,
        })
        if reported_at is not UNSET:
            field_dict["reported_at"] = reported_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        state = GetWorkerUpdatesResponse200WorkersItemState(d.pop("state"))




        version = d.pop("version")

        digest = d.pop("digest")

        cause_code = d.pop("cause_code")

        def _parse_reported_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                reported_at_type_0 = datetime.datetime.fromisoformat(data)



                return reported_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        reported_at = _parse_reported_at(d.pop("reported_at", UNSET))


        get_worker_updates_response_200_workers_item = cls(
            id=id,
            name=name,
            state=state,
            version=version,
            digest=digest,
            cause_code=cause_code,
            reported_at=reported_at,
        )

        return get_worker_updates_response_200_workers_item
