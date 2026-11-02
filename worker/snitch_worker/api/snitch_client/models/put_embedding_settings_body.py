from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PutEmbeddingSettingsBody")



@_attrs_define
class PutEmbeddingSettingsBody:
    """
        Attributes:
            url (str):
            model (str):
            validation_token (str):
     """

    url: str
    model: str
    validation_token: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        url = self.url

        model = self.model

        validation_token = self.validation_token


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "url": url,
            "model": model,
            "validation_token": validation_token,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        url = d.pop("url")

        model = d.pop("model")

        validation_token = d.pop("validation_token")

        put_embedding_settings_body = cls(
            url=url,
            model=model,
            validation_token=validation_token,
        )


        put_embedding_settings_body.additional_properties = d
        return put_embedding_settings_body

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
