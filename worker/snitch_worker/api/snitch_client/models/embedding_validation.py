from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="EmbeddingValidation")



@_attrs_define
class EmbeddingValidation:
    """
        Attributes:
            validation_token (str):
            dimensions (int):
            model (str):
            expires_at (datetime.datetime):
     """

    validation_token: str
    dimensions: int
    model: str
    expires_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        validation_token = self.validation_token

        dimensions = self.dimensions

        model = self.model

        expires_at = self.expires_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "validation_token": validation_token,
            "dimensions": dimensions,
            "model": model,
            "expires_at": expires_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        validation_token = d.pop("validation_token")

        dimensions = d.pop("dimensions")

        model = d.pop("model")

        expires_at = datetime.datetime.fromisoformat(d.pop("expires_at"))




        embedding_validation = cls(
            validation_token=validation_token,
            dimensions=dimensions,
            model=model,
            expires_at=expires_at,
        )


        embedding_validation.additional_properties = d
        return embedding_validation

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
