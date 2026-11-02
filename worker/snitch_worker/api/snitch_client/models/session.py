from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.user import User





T = TypeVar("T", bound="Session")



@_attrs_define
class Session:
    """
        Attributes:
            user (User):
            csrf_token (str):
            discord_linked (bool): Whether this account has a Discord identity linked.
            can_unlink_discord (bool): True only when Discord is linked and a local password is configured.
            discord_name (None | str | Unset): Display name of the linked Discord identity (global name, else username);
                null when unlinked or linked before it was kept. Read-only.
     """

    user: User
    csrf_token: str
    discord_linked: bool
    can_unlink_discord: bool
    discord_name: None | str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.user import User # noqa: PLC0415
        user = self.user.to_dict()

        csrf_token = self.csrf_token

        discord_linked = self.discord_linked

        can_unlink_discord = self.can_unlink_discord

        discord_name: None | str | Unset
        if isinstance(self.discord_name, Unset):
            discord_name = UNSET
        else:
            discord_name = self.discord_name


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "user": user,
            "csrf_token": csrf_token,
            "discord_linked": discord_linked,
            "can_unlink_discord": can_unlink_discord,
        })
        if discord_name is not UNSET:
            field_dict["discord_name"] = discord_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.user import User # noqa: PLC0415
        d = dict(src_dict)
        user = User.from_dict(d.pop("user"))




        csrf_token = d.pop("csrf_token")

        discord_linked = d.pop("discord_linked")

        can_unlink_discord = d.pop("can_unlink_discord")

        def _parse_discord_name(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        discord_name = _parse_discord_name(d.pop("discord_name", UNSET))


        session = cls(
            user=user,
            csrf_token=csrf_token,
            discord_linked=discord_linked,
            can_unlink_discord=can_unlink_discord,
            discord_name=discord_name,
        )


        session.additional_properties = d
        return session

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
