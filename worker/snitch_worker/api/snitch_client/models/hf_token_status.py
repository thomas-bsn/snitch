from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.hf_token_status_workers_item import HFTokenStatusWorkersItem





T = TypeVar("T", bound="HFTokenStatus")



@_attrs_define
class HFTokenStatus:
    """
        Attributes:
            configured (bool):
            fingerprint (str): Empreinte non réversible, vide si non configuré
            unreadable (bool): Valeur présente mais illisible avec la clé maître courante : à ressaisir
            workers (list[HFTokenStatusWorkersItem]):
     """

    configured: bool
    fingerprint: str
    unreadable: bool
    workers: list[HFTokenStatusWorkersItem]





    def to_dict(self) -> dict[str, Any]:
        from ..models.hf_token_status_workers_item import HFTokenStatusWorkersItem # noqa: PLC0415
        configured = self.configured

        fingerprint = self.fingerprint

        unreadable = self.unreadable

        workers = []
        for workers_item_data in self.workers:
            workers_item = workers_item_data.to_dict()
            workers.append(workers_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "configured": configured,
            "fingerprint": fingerprint,
            "unreadable": unreadable,
            "workers": workers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.hf_token_status_workers_item import HFTokenStatusWorkersItem # noqa: PLC0415
        d = dict(src_dict)
        configured = d.pop("configured")

        fingerprint = d.pop("fingerprint")

        unreadable = d.pop("unreadable")

        workers = []
        _workers = d.pop("workers")
        for workers_item_data in (_workers):
            workers_item = HFTokenStatusWorkersItem.from_dict(workers_item_data)



            workers.append(workers_item)


        hf_token_status = cls(
            configured=configured,
            fingerprint=fingerprint,
            unreadable=unreadable,
            workers=workers,
        )

        return hf_token_status
