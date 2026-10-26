from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.account_preferences_input_language import AccountPreferencesInputLanguage






T = TypeVar("T", bound="AccountPreferencesInput")



@_attrs_define
class AccountPreferencesInput:
    """
        Attributes:
            language (AccountPreferencesInputLanguage):
     """

    language: AccountPreferencesInputLanguage





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
        language = AccountPreferencesInputLanguage(d.pop("language"))




        account_preferences_input = cls(
            language=language,
        )

        return account_preferences_input
