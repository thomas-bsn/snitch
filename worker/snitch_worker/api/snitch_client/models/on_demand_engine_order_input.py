from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.on_demand_engine_order_input_orders_item import OnDemandEngineOrderInputOrdersItem





T = TypeVar("T", bound="OnDemandEngineOrderInput")



@_attrs_define
class OnDemandEngineOrderInput:
    """
        Attributes:
            family (str): Famille de bibliothèque de la table de Snitch (nemo).
            orders (list[OnDemandEngineOrderInputOrdersItem]):
            version (str | Unset): Version de la bibliothèque ; la version par défaut de la famille si omise.
            kind (str | Unset): Tâche couverte par l'engine (diarization, fingerprint pour NeMo) ; la première tâche de la
                famille si omise. Chaque tâche est une identité distincte.
     """

    family: str
    orders: list[OnDemandEngineOrderInputOrdersItem]
    version: str | Unset = UNSET
    kind: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.on_demand_engine_order_input_orders_item import OnDemandEngineOrderInputOrdersItem # noqa: PLC0415
        family = self.family

        orders = []
        for orders_item_data in self.orders:
            orders_item = orders_item_data.to_dict()
            orders.append(orders_item)



        version = self.version

        kind = self.kind


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "family": family,
            "orders": orders,
        })
        if version is not UNSET:
            field_dict["version"] = version
        if kind is not UNSET:
            field_dict["kind"] = kind

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.on_demand_engine_order_input_orders_item import OnDemandEngineOrderInputOrdersItem # noqa: PLC0415
        d = dict(src_dict)
        family = d.pop("family")

        orders = []
        _orders = d.pop("orders")
        for orders_item_data in (_orders):
            orders_item = OnDemandEngineOrderInputOrdersItem.from_dict(orders_item_data)



            orders.append(orders_item)


        version = d.pop("version", UNSET)

        kind = d.pop("kind", UNSET)

        on_demand_engine_order_input = cls(
            family=family,
            orders=orders,
            version=version,
            kind=kind,
        )

        return on_demand_engine_order_input
