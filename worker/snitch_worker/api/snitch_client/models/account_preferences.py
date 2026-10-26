from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.account_preferences_language import AccountPreferencesLanguage






T = TypeVar("T", bound="AccountPreferences")



@_attrs_define
class AccountPreferences:
    """
        Attributes:
            language (AccountPreferencesLanguage): Langue d'interface du compte ; null tant qu'aucun choix n'a été fait.
     """

    language: AccountPreferencesLanguage





    def to_dict(self) -> dict[str, Any]:
        language = self.language.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "language": language,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        language = AccountPreferencesLanguage(d.pop("language"))




        account_preferences = cls(
            language=language,
        )

        return account_preferences
