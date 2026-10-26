from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="EmbeddingSettings")



@_attrs_define
class EmbeddingSettings:
    """
        Attributes:
            url (str):
            model (str):
            url_from_environment (bool):
            model_from_environment (bool):
     """

    url: str
    model: str
    url_from_environment: bool
    model_from_environment: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        url = self.url

        model = self.model

        url_from_environment = self.url_from_environment

        model_from_environment = self.model_from_environment


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "url": url,
            "model": model,
            "url_from_environment": url_from_environment,
            "model_from_environment": model_from_environment,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        url = d.pop("url")

        model = d.pop("model")

        url_from_environment = d.pop("url_from_environment")

        model_from_environment = d.pop("model_from_environment")

        embedding_settings = cls(
            url=url,
            model=model,
            url_from_environment=url_from_environment,
            model_from_environment=model_from_environment,
        )


        embedding_settings.additional_properties = d
        return embedding_settings

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
