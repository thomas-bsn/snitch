from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.user_language import UserLanguage
from ..models.user_role import UserRole
from ..types import UNSET, Unset






T = TypeVar("T", bound="User")



@_attrs_define
class User:
    """
        Attributes:
            id (int):
            username (str):
            role (UserRole): super_admin : propriétaire de l'installation (compte créé par l'installation, unique, non
                rétrogradable, non supprimable) ; admin : Administrateur ; user : Utilisateur.
            approved (bool):
            language (UserLanguage | Unset): Langue d'interface choisie par la personne ; null tant qu'elle n'en a pas
                choisi.
            display_name (str | Unset): Pseudo choisi dans Snitch, sinon nom Discord pour les comptes générés, sinon
                identifiant local.
     """

    id: int
    username: str
    role: UserRole
    approved: bool
    language: UserLanguage | Unset = UNSET
    display_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        username = self.username

        role = self.role.value

        approved = self.approved

        language: str | Unset = UNSET
        if not isinstance(self.language, Unset):
            language = self.language.value


        display_name = self.display_name


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "username": username,
            "role": role,
            "approved": approved,
        })
        if language is not UNSET:
            field_dict["language"] = language
        if display_name is not UNSET:
            field_dict["display_name"] = display_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        username = d.pop("username")

        role = UserRole(d.pop("role"))




        approved = d.pop("approved")

        _language = d.pop("language", UNSET)
        language: UserLanguage | Unset
        if isinstance(_language,  Unset):
            language = UNSET
        else:
            language = UserLanguage(_language)




        display_name = d.pop("display_name", UNSET)

        user = cls(
            id=id,
            username=username,
            role=role,
            approved=approved,
            language=language,
            display_name=display_name,
        )


        user.additional_properties = d
        return user

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
