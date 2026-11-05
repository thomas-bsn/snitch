from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.member_role import MemberRole
from typing import cast
import datetime






T = TypeVar("T", bound="Member")



@_attrs_define
class Member:
    """
        Attributes:
            id (int):
            username (str):
            display_name (str): Nom affiché effectif : choisi, pseudo Discord ou identifiant.
            discord_linked (bool):
            role (MemberRole):
            created_at (datetime.datetime):
            clip_count (int):
     """

    id: int
    username: str
    display_name: str
    discord_linked: bool
    role: MemberRole
    created_at: datetime.datetime
    clip_count: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        username = self.username

        display_name = self.display_name

        discord_linked = self.discord_linked

        role = self.role.value

        created_at = self.created_at.isoformat()

        clip_count = self.clip_count


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "username": username,
            "display_name": display_name,
            "discord_linked": discord_linked,
            "role": role,
            "created_at": created_at,
            "clip_count": clip_count,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        username = d.pop("username")

        display_name = d.pop("display_name")

        discord_linked = d.pop("discord_linked")

        role = MemberRole(d.pop("role"))




        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        clip_count = d.pop("clip_count")

        member = cls(
            id=id,
            username=username,
            display_name=display_name,
            discord_linked=discord_linked,
            role=role,
            created_at=created_at,
            clip_count=clip_count,
        )


        member.additional_properties = d
        return member

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
