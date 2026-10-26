from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WorkerBundle")



@_attrs_define
class WorkerBundle:
    """ Compose et .env prêts à l'emploi, générés une seule fois avec le jeton ; jamais relus ensuite. Le .env contient le
    jeton.

        Attributes:
            compose (str):
            env (str):
            insecure_api_url (bool): Vrai si l'adresse publique n'est pas en HTTPS : le worker refuse le HTTP.
     """

    compose: str
    env: str
    insecure_api_url: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        compose = self.compose

        env = self.env

        insecure_api_url = self.insecure_api_url


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "compose": compose,
            "env": env,
            "insecure_api_url": insecure_api_url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        compose = d.pop("compose")

        env = d.pop("env")

        insecure_api_url = d.pop("insecure_api_url")

        worker_bundle = cls(
            compose=compose,
            env=env,
            insecure_api_url=insecure_api_url,
        )


        worker_bundle.additional_properties = d
        return worker_bundle

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
