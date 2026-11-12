from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.notification_webhook_input_categories_item import NotificationWebhookInputCategoriesItem
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="NotificationWebhookInput")



@_attrs_define
class NotificationWebhookInput:
    """
        Attributes:
            enabled (bool):
            categories (list[NotificationWebhookInputCategoriesItem]): Catégories relayées : access_requests (demandes
                d'accès), human_actions (voix à vérifier), worker_offline (Worker hors ligne), job_failures (traitement en
                échec, service ou modèle indisponible), job_blocked (traitement bloqué). Une décision sur une demande de dossier
                n'est jamais relayée.
            url (str | Unset): Adresse https://discord.com/api/webhooks/<id>/<jeton>. Absente : l'adresse enregistrée est
                conservée.
     """

    enabled: bool
    categories: list[NotificationWebhookInputCategoriesItem]
    url: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        enabled = self.enabled

        categories = []
        for categories_item_data in self.categories:
            categories_item = categories_item_data.value
            categories.append(categories_item)



        url = self.url


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "enabled": enabled,
            "categories": categories,
        })
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        enabled = d.pop("enabled")

        categories = []
        _categories = d.pop("categories")
        for categories_item_data in (_categories):
            categories_item = NotificationWebhookInputCategoriesItem(categories_item_data)



            categories.append(categories_item)


        url = d.pop("url", UNSET)

        notification_webhook_input = cls(
            enabled=enabled,
            categories=categories,
            url=url,
        )


        notification_webhook_input.additional_properties = d
        return notification_webhook_input

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
