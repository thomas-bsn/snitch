from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.runtime_binding_pending_intent import RuntimeBindingPendingIntent
from ..models.runtime_binding_pending_status import RuntimeBindingPendingStatus






T = TypeVar("T", bound="RuntimeBindingPending")



@_attrs_define
class RuntimeBindingPending:
    """
        Attributes:
            order_id (str):
            intent (RuntimeBindingPendingIntent):
            digest (str):
            version (str):
            status (RuntimeBindingPendingStatus):
     """

    order_id: str
    intent: RuntimeBindingPendingIntent
    digest: str
    version: str
    status: RuntimeBindingPendingStatus
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        order_id = self.order_id

        intent = self.intent.value

        digest = self.digest

        version = self.version

        status = self.status.value


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "order_id": order_id,
            "intent": intent,
            "digest": digest,
            "version": version,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        order_id = d.pop("order_id")

        intent = RuntimeBindingPendingIntent(d.pop("intent"))




        digest = d.pop("digest")

        version = d.pop("version")

        status = RuntimeBindingPendingStatus(d.pop("status"))




        runtime_binding_pending = cls(
            order_id=order_id,
            intent=intent,
            digest=digest,
            version=version,
            status=status,
        )


        runtime_binding_pending.additional_properties = d
        return runtime_binding_pending

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
