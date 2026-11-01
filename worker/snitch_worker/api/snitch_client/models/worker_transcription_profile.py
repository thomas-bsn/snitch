from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WorkerTranscriptionProfile")



@_attrs_define
class WorkerTranscriptionProfile:
    """
        Attributes:
            model (str):
            language (str): Code de langue Whisper : l’un des 100 codes que Whisper accepte (liste fermée, codes de Whisper
                tels quels, par exemple jw pour le javanais, haw, yue).
     """

    model: str
    language: str





    def to_dict(self) -> dict[str, Any]:
        model = self.model

        language = self.language


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "model": model,
            "language": language,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        model = d.pop("model")

        language = d.pop("language")

        worker_transcription_profile = cls(
            model=model,
            language=language,
        )

        return worker_transcription_profile
