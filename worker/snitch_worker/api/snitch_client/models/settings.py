from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="Settings")



@_attrs_define
class Settings:
    """
        Attributes:
            client_id (str):
            secret_configured (bool): Whether a nonempty secret is stored. The secret itself is never returned.
            registration_approval (bool):
            callback_url (str): Discord return address to register, built from the public address of the request (or the
                configured PUBLIC_URL).
     """

    client_id: str
    secret_configured: bool
    registration_approval: bool
    callback_url: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        client_id = self.client_id

        secret_configured = self.secret_configured

        registration_approval = self.registration_approval

        callback_url = self.callback_url


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "client_id": client_id,
            "secret_configured": secret_configured,
            "registration_approval": registration_approval,
            "callback_url": callback_url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        client_id = d.pop("client_id")

        secret_configured = d.pop("secret_configured")

        registration_approval = d.pop("registration_approval")

        callback_url = d.pop("callback_url")

        settings = cls(
            client_id=client_id,
            secret_configured=secret_configured,
            registration_approval=registration_approval,
            callback_url=callback_url,
        )


        settings.additional_properties = d
        return settings

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
