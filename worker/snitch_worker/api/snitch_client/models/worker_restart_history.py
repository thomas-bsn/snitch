from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.worker_restart_history_items_item import WorkerRestartHistoryItemsItem





T = TypeVar("T", bound="WorkerRestartHistory")



@_attrs_define
class WorkerRestartHistory:
    """
        Attributes:
            items (list[WorkerRestartHistoryItemsItem]):
            supervisor_enrolled (bool):
            supervisor_last_seen_at (datetime.datetime | None):
            supervisor_available (bool):
     """

    items: list[WorkerRestartHistoryItemsItem]
    supervisor_enrolled: bool
    supervisor_last_seen_at: datetime.datetime | None
    supervisor_available: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_restart_history_items_item import WorkerRestartHistoryItemsItem # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        supervisor_enrolled = self.supervisor_enrolled

        supervisor_last_seen_at: None | str
        if isinstance(self.supervisor_last_seen_at, datetime.datetime):
            supervisor_last_seen_at = self.supervisor_last_seen_at.isoformat()
        else:
            supervisor_last_seen_at = self.supervisor_last_seen_at

        supervisor_available = self.supervisor_available


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "items": items,
            "supervisor_enrolled": supervisor_enrolled,
            "supervisor_last_seen_at": supervisor_last_seen_at,
            "supervisor_available": supervisor_available,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_restart_history_items_item import WorkerRestartHistoryItemsItem # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = WorkerRestartHistoryItemsItem.from_dict(items_item_data)



            items.append(items_item)


        supervisor_enrolled = d.pop("supervisor_enrolled")

        def _parse_supervisor_last_seen_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                supervisor_last_seen_at_type_0 = datetime.datetime.fromisoformat(data)



                return supervisor_last_seen_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        supervisor_last_seen_at = _parse_supervisor_last_seen_at(d.pop("supervisor_last_seen_at"))


        supervisor_available = d.pop("supervisor_available")

        worker_restart_history = cls(
            items=items,
            supervisor_enrolled=supervisor_enrolled,
            supervisor_last_seen_at=supervisor_last_seen_at,
            supervisor_available=supervisor_available,
        )


        worker_restart_history.additional_properties = d
        return worker_restart_history

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
