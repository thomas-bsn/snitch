from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.settings_master_key_origin import SettingsMasterKeyOrigin
from ..models.settings_unreadable_secrets_item import SettingsUnreadableSecretsItem
from typing import cast






T = TypeVar("T", bound="Settings")



@_attrs_define
class Settings:
    """
        Attributes:
            client_id (str):
            secret_configured (bool): Whether a nonempty secret is stored. The secret itself is never returned.
            registration_approval (bool):
            master_key_origin (SettingsMasterKeyOrigin): provided : la clé vient de la variable MASTER_KEY ; generated :
                l'application l'a générée et la conserve dans son volume de secrets.
            unreadable_secrets (list[SettingsUnreadableSecretsItem]): Réglages chiffrés que la clé maître actuelle ne
                déchiffre plus (clé absente ou différente). Ils sont traités comme non configurés et sont à ressaisir ;
                l'application démarre quand même.
     """

    client_id: str
    secret_configured: bool
    registration_approval: bool
    master_key_origin: SettingsMasterKeyOrigin
    unreadable_secrets: list[SettingsUnreadableSecretsItem]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        client_id = self.client_id

        secret_configured = self.secret_configured

        registration_approval = self.registration_approval

        master_key_origin = self.master_key_origin.value

        unreadable_secrets = []
        for unreadable_secrets_item_data in self.unreadable_secrets:
            unreadable_secrets_item = unreadable_secrets_item_data.value
            unreadable_secrets.append(unreadable_secrets_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "client_id": client_id,
            "secret_configured": secret_configured,
            "registration_approval": registration_approval,
            "master_key_origin": master_key_origin,
            "unreadable_secrets": unreadable_secrets,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        client_id = d.pop("client_id")

        secret_configured = d.pop("secret_configured")

        registration_approval = d.pop("registration_approval")

        master_key_origin = SettingsMasterKeyOrigin(d.pop("master_key_origin"))




        unreadable_secrets = []
        _unreadable_secrets = d.pop("unreadable_secrets")
        for unreadable_secrets_item_data in (_unreadable_secrets):
            unreadable_secrets_item = SettingsUnreadableSecretsItem(unreadable_secrets_item_data)



            unreadable_secrets.append(unreadable_secrets_item)


        settings = cls(
            client_id=client_id,
            secret_configured=secret_configured,
            registration_approval=registration_approval,
            master_key_origin=master_key_origin,
            unreadable_secrets=unreadable_secrets,
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
