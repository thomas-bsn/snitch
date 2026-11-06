from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_model_add_kind import VoiceModelAddKind






T = TypeVar("T", bound="VoiceModelAdd")



@_attrs_define
class VoiceModelAdd:
    """
        Attributes:
            source (str):
            revision (str): Commit montré par l'inspection.
            kind (VoiceModelAddKind):
     """

    source: str
    revision: str
    kind: VoiceModelAddKind





    def to_dict(self) -> dict[str, Any]:
        source = self.source

        revision = self.revision

        kind = self.kind.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "source": source,
            "revision": revision,
            "kind": kind,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        source = d.pop("source")

        revision = d.pop("revision")

        kind = VoiceModelAddKind(d.pop("kind"))




        voice_model_add = cls(
            source=source,
            revision=revision,
            kind=kind,
        )

        return voice_model_add
