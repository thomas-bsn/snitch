from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoicePipelineFake")



@_attrs_define
class VoicePipelineFake:
    """ Réglages du fournisseur factice déterministe, réservés aux tests.

        Attributes:
            speakers (int):
            dimensions (int):
     """

    speakers: int
    dimensions: int





    def to_dict(self) -> dict[str, Any]:
        speakers = self.speakers

        dimensions = self.dimensions


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "speakers": speakers,
            "dimensions": dimensions,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        speakers = d.pop("speakers")

        dimensions = d.pop("dimensions")

        voice_pipeline_fake = cls(
            speakers=speakers,
            dimensions=dimensions,
        )

        return voice_pipeline_fake
