from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.member_detail_language import MemberDetailLanguage
from ..models.member_detail_role import MemberDetailRole
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.member_permissions import MemberPermissions





T = TypeVar("T", bound="MemberDetail")



@_attrs_define
class MemberDetail:
    """
        Attributes:
            id (int):
            username (str):
            display_name (None | str):
            language (MemberDetailLanguage):
            role (MemberDetailRole):
            approved (bool):
            discord_linked (bool):
            discord_username (None | str):
            has_password (bool):
            created_at (datetime.datetime):
            clip_count (int):
            session_count (int):
            permissions (MemberPermissions):
     """

    id: int
    username: str
    display_name: None | str
    language: MemberDetailLanguage
    role: MemberDetailRole
    approved: bool
    discord_linked: bool
    discord_username: None | str
    has_password: bool
    created_at: datetime.datetime
    clip_count: int
    session_count: int
    permissions: MemberPermissions
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.member_permissions import MemberPermissions # noqa: PLC0415
        id = self.id

        username = self.username

        display_name: None | str
        display_name = self.display_name

        language = self.language.value

        role = self.role.value

        approved = self.approved

        discord_linked = self.discord_linked

        discord_username: None | str
        discord_username = self.discord_username

        has_password = self.has_password

        created_at = self.created_at.isoformat()

        clip_count = self.clip_count

        session_count = self.session_count

        permissions = self.permissions.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "username": username,
            "display_name": display_name,
            "language": language,
            "role": role,
            "approved": approved,
            "discord_linked": discord_linked,
            "discord_username": discord_username,
            "has_password": has_password,
            "created_at": created_at,
            "clip_count": clip_count,
            "session_count": session_count,
            "permissions": permissions,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.member_permissions import MemberPermissions # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        username = d.pop("username")

        def _parse_display_name(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        display_name = _parse_display_name(d.pop("display_name"))


        language = MemberDetailLanguage(d.pop("language"))




        role = MemberDetailRole(d.pop("role"))




        approved = d.pop("approved")

        discord_linked = d.pop("discord_linked")

        def _parse_discord_username(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        discord_username = _parse_discord_username(d.pop("discord_username"))


        has_password = d.pop("has_password")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        clip_count = d.pop("clip_count")

        session_count = d.pop("session_count")

        permissions = MemberPermissions.from_dict(d.pop("permissions"))




        member_detail = cls(
            id=id,
            username=username,
            display_name=display_name,
            language=language,
            role=role,
            approved=approved,
            discord_linked=discord_linked,
            discord_username=discord_username,
            has_password=has_password,
            created_at=created_at,
            clip_count=clip_count,
            session_count=session_count,
            permissions=permissions,
        )


        member_detail.additional_properties = d
        return member_detail

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
