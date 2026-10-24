from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SettingsInput")



@_attrs_define
class SettingsInput:
    """ 
        Attributes:
            client_id (str): Required application identifier. Empty disables Discord and removes the stored secret.
                Whitespace-only values are rejected; saving does not verify credentials with Discord.
            registration_approval (bool): Booléen obligatoire et non nul. Politique commune aux nouvelles inscriptions
                locales et Discord, sans approbation rétroactive des comptes déjà en attente.
            client_secret (str | Unset): Write-only application secret. Omit or send an empty string to retain the current
                secret. Enabling Discord requires a new or existing nonempty secret; whitespace-only values are rejected.
     """

    client_id: str
    registration_approval: bool
    client_secret: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        client_id = self.client_id

        registration_approval = self.registration_approval

        client_secret = self.client_secret


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "client_id": client_id,
            "registration_approval": registration_approval,
        })
        if client_secret is not UNSET:
            field_dict["client_secret"] = client_secret

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        client_id = d.pop("client_id")

        registration_approval = d.pop("registration_approval")

        client_secret = d.pop("client_secret", UNSET)

        settings_input = cls(
            client_id=client_id,
            registration_approval=registration_approval,
            client_secret=client_secret,
        )

        return settings_input

