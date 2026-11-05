from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.discord_bot_status_status import DiscordBotStatusStatus
from typing import cast
import datetime






T = TypeVar("T", bound="DiscordBotStatus")



@_attrs_define
class DiscordBotStatus:
    """
        Attributes:
            status (DiscordBotStatusStatus):
            bot_name (None | str):
            last_error (None | str):
            last_sync_at (datetime.datetime | None):
            last_refreshed (int):
            last_failed (int):
            running (bool):
     """

    status: DiscordBotStatusStatus
    bot_name: None | str
    last_error: None | str
    last_sync_at: datetime.datetime | None
    last_refreshed: int
    last_failed: int
    running: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        status = self.status.value

        bot_name: None | str
        bot_name = self.bot_name

        last_error: None | str
        last_error = self.last_error

        last_sync_at: None | str
        if isinstance(self.last_sync_at, datetime.datetime):
            last_sync_at = self.last_sync_at.isoformat()
        else:
            last_sync_at = self.last_sync_at

        last_refreshed = self.last_refreshed

        last_failed = self.last_failed

        running = self.running


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "status": status,
            "bot_name": bot_name,
            "last_error": last_error,
            "last_sync_at": last_sync_at,
            "last_refreshed": last_refreshed,
            "last_failed": last_failed,
            "running": running,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = DiscordBotStatusStatus(d.pop("status"))




        def _parse_bot_name(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        bot_name = _parse_bot_name(d.pop("bot_name"))


        def _parse_last_error(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        last_error = _parse_last_error(d.pop("last_error"))


        def _parse_last_sync_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_sync_at_type_0 = datetime.datetime.fromisoformat(data)



                return last_sync_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        last_sync_at = _parse_last_sync_at(d.pop("last_sync_at"))


        last_refreshed = d.pop("last_refreshed")

        last_failed = d.pop("last_failed")

        running = d.pop("running")

        discord_bot_status = cls(
            status=status,
            bot_name=bot_name,
            last_error=last_error,
            last_sync_at=last_sync_at,
            last_refreshed=last_refreshed,
            last_failed=last_failed,
            running=running,
        )


        discord_bot_status.additional_properties = d
        return discord_bot_status

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
