from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.member_patch_language import MemberPatchLanguage
from ..models.member_patch_role import MemberPatchRole
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="MemberPatch")



@_attrs_define
class MemberPatch:
    """
        Attributes:
            username (str | Unset):
            display_name (None | str | Unset):
            language (MemberPatchLanguage | Unset):
            role (MemberPatchRole | Unset):
            approved (bool | Unset):
     """

    username: str | Unset = UNSET
    display_name: None | str | Unset = UNSET
    language: MemberPatchLanguage | Unset = UNSET
    role: MemberPatchRole | Unset = UNSET
    approved: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        username = self.username

        display_name: None | str | Unset
        if isinstance(self.display_name, Unset):
            display_name = UNSET
        else:
            display_name = self.display_name

        language: str | Unset = UNSET
        if not isinstance(self.language, Unset):
            language = self.language.value


        role: str | Unset = UNSET
        if not isinstance(self.role, Unset):
            role = self.role.value


        approved = self.approved


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if username is not UNSET:
            field_dict["username"] = username
        if display_name is not UNSET:
            field_dict["display_name"] = display_name
        if language is not UNSET:
            field_dict["language"] = language
        if role is not UNSET:
            field_dict["role"] = role
        if approved is not UNSET:
            field_dict["approved"] = approved

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        username = d.pop("username", UNSET)

        def _parse_display_name(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        display_name = _parse_display_name(d.pop("display_name", UNSET))


        _language = d.pop("language", UNSET)
        language: MemberPatchLanguage | Unset
        if isinstance(_language,  Unset):
            language = UNSET
        else:
            language = MemberPatchLanguage(_language)




        _role = d.pop("role", UNSET)
        role: MemberPatchRole | Unset
        if isinstance(_role,  Unset):
            role = UNSET
        else:
            role = MemberPatchRole(_role)




        approved = d.pop("approved", UNSET)

        member_patch = cls(
            username=username,
            display_name=display_name,
            language=language,
            role=role,
            approved=approved,
        )

        return member_patch
