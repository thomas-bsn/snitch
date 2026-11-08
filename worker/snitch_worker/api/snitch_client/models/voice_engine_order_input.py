from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_engine_order_input_intent import VoiceEngineOrderInputIntent
from typing import cast

if TYPE_CHECKING:
  from ..models.voice_engine_order_input_orders_item import VoiceEngineOrderInputOrdersItem





T = TypeVar("T", bound="VoiceEngineOrderInput")



@_attrs_define
class VoiceEngineOrderInput:
    """
        Attributes:
            digest (str): Identité signée de la version approuvée (empreinte du registre).
            intent (VoiceEngineOrderInputIntent):
            orders (list[VoiceEngineOrderInputOrdersItem]):
     """

    digest: str
    intent: VoiceEngineOrderInputIntent
    orders: list[VoiceEngineOrderInputOrdersItem]





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_engine_order_input_orders_item import VoiceEngineOrderInputOrdersItem # noqa: PLC0415
        digest = self.digest

        intent = self.intent.value

        orders = []
        for orders_item_data in self.orders:
            orders_item = orders_item_data.to_dict()
            orders.append(orders_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "digest": digest,
            "intent": intent,
            "orders": orders,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_engine_order_input_orders_item import VoiceEngineOrderInputOrdersItem # noqa: PLC0415
        d = dict(src_dict)
        digest = d.pop("digest")

        intent = VoiceEngineOrderInputIntent(d.pop("intent"))




        orders = []
        _orders = d.pop("orders")
        for orders_item_data in (_orders):
            orders_item = VoiceEngineOrderInputOrdersItem.from_dict(orders_item_data)



            orders.append(orders_item)


        voice_engine_order_input = cls(
            digest=digest,
            intent=intent,
            orders=orders,
        )

        return voice_engine_order_input
