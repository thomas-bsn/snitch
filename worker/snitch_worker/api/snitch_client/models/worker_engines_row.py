from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_engine import WorkerEngine





T = TypeVar("T", bound="WorkerEnginesRow")



@_attrs_define
class WorkerEnginesRow:
    """
        Attributes:
            worker_id (str):
            name (str):
            connected (bool):
            engines (list[WorkerEngine]):
     """

    worker_id: str
    name: str
    connected: bool
    engines: list[WorkerEngine]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_engine import WorkerEngine # noqa: PLC0415
        worker_id = self.worker_id

        name = self.name

        connected = self.connected

        engines = []
        for engines_item_data in self.engines:
            engines_item = engines_item_data.to_dict()
            engines.append(engines_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "worker_id": worker_id,
            "name": name,
            "connected": connected,
            "engines": engines,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_engine import WorkerEngine # noqa: PLC0415
        d = dict(src_dict)
        worker_id = d.pop("worker_id")

        name = d.pop("name")

        connected = d.pop("connected")

        engines = []
        _engines = d.pop("engines")
        for engines_item_data in (_engines):
            engines_item = WorkerEngine.from_dict(engines_item_data)



            engines.append(engines_item)


        worker_engines_row = cls(
            worker_id=worker_id,
            name=name,
            connected=connected,
            engines=engines,
        )


        worker_engines_row.additional_properties = d
        return worker_engines_row

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
