from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.notification_webhook_audit_item_action import NotificationWebhookAuditItemAction
from ..models.notification_webhook_audit_item_categories_item import NotificationWebhookAuditItemCategoriesItem
from typing import cast
import datetime






T = TypeVar("T", bound="NotificationWebhookAuditItem")



@_attrs_define
class NotificationWebhookAuditItem:
    """
        Attributes:
            action (NotificationWebhookAuditItemAction): auto_disabled : Discord a répondu que le webhook n'existe plus ;
                aucun auteur.
            enabled (bool):
            categories (list[NotificationWebhookAuditItemCategoriesItem]):
            actor_username (str):
            created_at (datetime.datetime):
     """

    action: NotificationWebhookAuditItemAction
    enabled: bool
    categories: list[NotificationWebhookAuditItemCategoriesItem]
    actor_username: str
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        action = self.action.value

        enabled = self.enabled

        categories = []
        for categories_item_data in self.categories:
            categories_item = categories_item_data.value
            categories.append(categories_item)



        actor_username = self.actor_username

        created_at = self.created_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "action": action,
            "enabled": enabled,
            "categories": categories,
            "actor_username": actor_username,
            "created_at": created_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = NotificationWebhookAuditItemAction(d.pop("action"))




        enabled = d.pop("enabled")

        categories = []
        _categories = d.pop("categories")
        for categories_item_data in (_categories):
            categories_item = NotificationWebhookAuditItemCategoriesItem(categories_item_data)



            categories.append(categories_item)


        actor_username = d.pop("actor_username")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        notification_webhook_audit_item = cls(
            action=action,
            enabled=enabled,
            categories=categories,
            actor_username=actor_username,
            created_at=created_at,
        )


        notification_webhook_audit_item.additional_properties = d
        return notification_webhook_audit_item

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
