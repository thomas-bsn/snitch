from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.notification_webhook_deliveries_item_category import NotificationWebhookDeliveriesItemCategory
from ..models.notification_webhook_deliveries_item_status import NotificationWebhookDeliveriesItemStatus
from typing import cast
import datetime






T = TypeVar("T", bound="NotificationWebhookDeliveriesItem")



@_attrs_define
class NotificationWebhookDeliveriesItem:
    """
        Attributes:
            id (int):
            category (NotificationWebhookDeliveriesItemCategory):
            status (NotificationWebhookDeliveriesItemStatus):
            attempts (int):
            last_error (str):
            created_at (datetime.datetime):
            next_attempt_at (datetime.datetime):
            sent_at (datetime.datetime | None):
     """

    id: int
    category: NotificationWebhookDeliveriesItemCategory
    status: NotificationWebhookDeliveriesItemStatus
    attempts: int
    last_error: str
    created_at: datetime.datetime
    next_attempt_at: datetime.datetime
    sent_at: datetime.datetime | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        category = self.category.value

        status = self.status.value

        attempts = self.attempts

        last_error = self.last_error

        created_at = self.created_at.isoformat()

        next_attempt_at = self.next_attempt_at.isoformat()

        sent_at: None | str
        if isinstance(self.sent_at, datetime.datetime):
            sent_at = self.sent_at.isoformat()
        else:
            sent_at = self.sent_at


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "category": category,
            "status": status,
            "attempts": attempts,
            "last_error": last_error,
            "created_at": created_at,
            "next_attempt_at": next_attempt_at,
            "sent_at": sent_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        category = NotificationWebhookDeliveriesItemCategory(d.pop("category"))




        status = NotificationWebhookDeliveriesItemStatus(d.pop("status"))




        attempts = d.pop("attempts")

        last_error = d.pop("last_error")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        next_attempt_at = datetime.datetime.fromisoformat(d.pop("next_attempt_at"))




        def _parse_sent_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                sent_at_type_0 = datetime.datetime.fromisoformat(data)



                return sent_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        sent_at = _parse_sent_at(d.pop("sent_at"))


        notification_webhook_deliveries_item = cls(
            id=id,
            category=category,
            status=status,
            attempts=attempts,
            last_error=last_error,
            created_at=created_at,
            next_attempt_at=next_attempt_at,
            sent_at=sent_at,
        )


        notification_webhook_deliveries_item.additional_properties = d
        return notification_webhook_deliveries_item

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
