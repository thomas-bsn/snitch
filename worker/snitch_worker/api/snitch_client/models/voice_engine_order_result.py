from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_engine_order_result_status import VoiceEngineOrderResultStatus
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.engine_blockers import EngineBlockers
  from ..models.engine_order import EngineOrder





T = TypeVar("T", bound="VoiceEngineOrderResult")



@_attrs_define
class VoiceEngineOrderResult:
    """
        Attributes:
            worker_id (str):
            status (VoiceEngineOrderResultStatus):
            cause_code (str | Unset): Cause stable d'un refus (engine_not_found, engine_release_unavailable,
                engine_order_in_progress, engine_already_installed, engine_not_installed, engine_in_use, idempotency_key_reused,
                worker_not_found, invalid_idempotency_key).
            order (EngineOrder | Unset):
            blockers (EngineBlockers | Unset):
     """

    worker_id: str
    status: VoiceEngineOrderResultStatus
    cause_code: str | Unset = UNSET
    order: EngineOrder | Unset = UNSET
    blockers: EngineBlockers | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.engine_blockers import EngineBlockers # noqa: PLC0415
        from ..models.engine_order import EngineOrder # noqa: PLC0415
        worker_id = self.worker_id

        status = self.status.value

        cause_code = self.cause_code

        order: dict[str, Any] | Unset = UNSET
        if not isinstance(self.order, Unset):
            order = self.order.to_dict()

        blockers: dict[str, Any] | Unset = UNSET
        if not isinstance(self.blockers, Unset):
            blockers = self.blockers.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "worker_id": worker_id,
            "status": status,
        })
        if cause_code is not UNSET:
            field_dict["cause_code"] = cause_code
        if order is not UNSET:
            field_dict["order"] = order
        if blockers is not UNSET:
            field_dict["blockers"] = blockers

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.engine_blockers import EngineBlockers # noqa: PLC0415
        from ..models.engine_order import EngineOrder # noqa: PLC0415
        d = dict(src_dict)
        worker_id = d.pop("worker_id")

        status = VoiceEngineOrderResultStatus(d.pop("status"))




        cause_code = d.pop("cause_code", UNSET)

        _order = d.pop("order", UNSET)
        order: EngineOrder | Unset
        if isinstance(_order,  Unset):
            order = UNSET
        else:
            order = EngineOrder.from_dict(_order)




        _blockers = d.pop("blockers", UNSET)
        blockers: EngineBlockers | Unset
        if isinstance(_blockers,  Unset):
            blockers = UNSET
        else:
            blockers = EngineBlockers.from_dict(_blockers)




        voice_engine_order_result = cls(
            worker_id=worker_id,
            status=status,
            cause_code=cause_code,
            order=order,
            blockers=blockers,
        )


        voice_engine_order_result.additional_properties = d
        return voice_engine_order_result

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
