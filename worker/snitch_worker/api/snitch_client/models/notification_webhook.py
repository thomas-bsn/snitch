from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.notification_webhook_categories_item import NotificationWebhookCategoriesItem
from ..models.notification_webhook_last_status import NotificationWebhookLastStatus
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.notification_webhook_audit_item import NotificationWebhookAuditItem
  from ..models.notification_webhook_deliveries_item import NotificationWebhookDeliveriesItem





T = TypeVar("T", bound="NotificationWebhook")



@_attrs_define
class NotificationWebhook:
    """
        Attributes:
            configured (bool): Une adresse est enregistrée (chiffrée) et lisible avec la clé maître actuelle.
            enabled (bool):
            categories (list[NotificationWebhookCategoriesItem]):
            hint (str): Quatre derniers chiffres de l'identifiant du webhook, jamais le jeton.
            last_status (NotificationWebhookLastStatus):
            last_error (str): Code normalisé (timeout, network, rate_limited, webhook_rejected, http_<statut>), jamais
                l'erreur brute.
            last_attempt_at (datetime.datetime | None):
            last_success_at (datetime.datetime | None):
            pending (int): Livraisons en attente ou en nouvel essai.
            failed (int): Livraisons abandonnées sur les sept derniers jours.
            deliveries (list[NotificationWebhookDeliveriesItem]): Vingt dernières livraisons.
            audit (list[NotificationWebhookAuditItem]): Vingt derniers changements, immuables ; l'adresse n'y figure jamais.
     """

    configured: bool
    enabled: bool
    categories: list[NotificationWebhookCategoriesItem]
    hint: str
    last_status: NotificationWebhookLastStatus
    last_error: str
    last_attempt_at: datetime.datetime | None
    last_success_at: datetime.datetime | None
    pending: int
    failed: int
    deliveries: list[NotificationWebhookDeliveriesItem]
    audit: list[NotificationWebhookAuditItem]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.notification_webhook_audit_item import NotificationWebhookAuditItem # noqa: PLC0415
        from ..models.notification_webhook_deliveries_item import NotificationWebhookDeliveriesItem # noqa: PLC0415
        configured = self.configured

        enabled = self.enabled

        categories = []
        for categories_item_data in self.categories:
            categories_item = categories_item_data.value
            categories.append(categories_item)



        hint = self.hint

        last_status = self.last_status.value

        last_error = self.last_error

        last_attempt_at: None | str
        if isinstance(self.last_attempt_at, datetime.datetime):
            last_attempt_at = self.last_attempt_at.isoformat()
        else:
            last_attempt_at = self.last_attempt_at

        last_success_at: None | str
        if isinstance(self.last_success_at, datetime.datetime):
            last_success_at = self.last_success_at.isoformat()
        else:
            last_success_at = self.last_success_at

        pending = self.pending

        failed = self.failed

        deliveries = []
        for deliveries_item_data in self.deliveries:
            deliveries_item = deliveries_item_data.to_dict()
            deliveries.append(deliveries_item)



        audit = []
        for audit_item_data in self.audit:
            audit_item = audit_item_data.to_dict()
            audit.append(audit_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "configured": configured,
            "enabled": enabled,
            "categories": categories,
            "hint": hint,
            "last_status": last_status,
            "last_error": last_error,
            "last_attempt_at": last_attempt_at,
            "last_success_at": last_success_at,
            "pending": pending,
            "failed": failed,
            "deliveries": deliveries,
            "audit": audit,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.notification_webhook_audit_item import NotificationWebhookAuditItem # noqa: PLC0415
        from ..models.notification_webhook_deliveries_item import NotificationWebhookDeliveriesItem # noqa: PLC0415
        d = dict(src_dict)
        configured = d.pop("configured")

        enabled = d.pop("enabled")

        categories = []
        _categories = d.pop("categories")
        for categories_item_data in (_categories):
            categories_item = NotificationWebhookCategoriesItem(categories_item_data)



            categories.append(categories_item)


        hint = d.pop("hint")

        last_status = NotificationWebhookLastStatus(d.pop("last_status"))




        last_error = d.pop("last_error")

        def _parse_last_attempt_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_attempt_at_type_0 = datetime.datetime.fromisoformat(data)



                return last_attempt_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        last_attempt_at = _parse_last_attempt_at(d.pop("last_attempt_at"))


        def _parse_last_success_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_success_at_type_0 = datetime.datetime.fromisoformat(data)



                return last_success_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        last_success_at = _parse_last_success_at(d.pop("last_success_at"))


        pending = d.pop("pending")

        failed = d.pop("failed")

        deliveries = []
        _deliveries = d.pop("deliveries")
        for deliveries_item_data in (_deliveries):
            deliveries_item = NotificationWebhookDeliveriesItem.from_dict(deliveries_item_data)



            deliveries.append(deliveries_item)


        audit = []
        _audit = d.pop("audit")
        for audit_item_data in (_audit):
            audit_item = NotificationWebhookAuditItem.from_dict(audit_item_data)



            audit.append(audit_item)


        notification_webhook = cls(
            configured=configured,
            enabled=enabled,
            categories=categories,
            hint=hint,
            last_status=last_status,
            last_error=last_error,
            last_attempt_at=last_attempt_at,
            last_success_at=last_success_at,
            pending=pending,
            failed=failed,
            deliveries=deliveries,
            audit=audit,
        )


        notification_webhook.additional_properties = d
        return notification_webhook

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
