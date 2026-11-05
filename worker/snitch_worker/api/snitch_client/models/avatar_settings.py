from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.avatar_settings_source import AvatarSettingsSource
from typing import cast
import datetime






T = TypeVar("T", bound="AvatarSettings")



@_attrs_define
class AvatarSettings:
    """
        Attributes:
            source (AvatarSettingsSource):
            discord_linked (bool):
            has_custom (bool):
            has_discord (bool):
            bot_configured (bool):
            auto_sync (bool): Vrai quand la photo affichée suit Discord et que le bot est configuré.
            last_synced_at (datetime.datetime | None):
            last_error (None | str):
     """

    source: AvatarSettingsSource
    discord_linked: bool
    has_custom: bool
    has_discord: bool
    bot_configured: bool
    auto_sync: bool
    last_synced_at: datetime.datetime | None
    last_error: None | str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        source = self.source.value

        discord_linked = self.discord_linked

        has_custom = self.has_custom

        has_discord = self.has_discord

        bot_configured = self.bot_configured

        auto_sync = self.auto_sync

        last_synced_at: None | str
        if isinstance(self.last_synced_at, datetime.datetime):
            last_synced_at = self.last_synced_at.isoformat()
        else:
            last_synced_at = self.last_synced_at

        last_error: None | str
        last_error = self.last_error


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "source": source,
            "discord_linked": discord_linked,
            "has_custom": has_custom,
            "has_discord": has_discord,
            "bot_configured": bot_configured,
            "auto_sync": auto_sync,
            "last_synced_at": last_synced_at,
            "last_error": last_error,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        source = AvatarSettingsSource(d.pop("source"))




        discord_linked = d.pop("discord_linked")

        has_custom = d.pop("has_custom")

        has_discord = d.pop("has_discord")

        bot_configured = d.pop("bot_configured")

        auto_sync = d.pop("auto_sync")

        def _parse_last_synced_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_synced_at_type_0 = datetime.datetime.fromisoformat(data)



                return last_synced_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        last_synced_at = _parse_last_synced_at(d.pop("last_synced_at"))


        def _parse_last_error(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        last_error = _parse_last_error(d.pop("last_error"))


        avatar_settings = cls(
            source=source,
            discord_linked=discord_linked,
            has_custom=has_custom,
            has_discord=has_discord,
            bot_configured=bot_configured,
            auto_sync=auto_sync,
            last_synced_at=last_synced_at,
            last_error=last_error,
        )


        avatar_settings.additional_properties = d
        return avatar_settings

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
