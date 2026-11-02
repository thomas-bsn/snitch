from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="VoiceHistoryEntry")



@_attrs_define
class VoiceHistoryEntry:
    """ Résultat de diarisation remplacé par une relance, avec sa provenance.

        Attributes:
            job_id (int):
            model (str):
            model_version (str):
            completed_at (datetime.datetime):
            speaker_count (int):
            validated_speakers (int): Locuteurs validés de ce résultat, conservés avec leurs échantillons vocaux.
     """

    job_id: int
    model: str
    model_version: str
    completed_at: datetime.datetime
    speaker_count: int
    validated_speakers: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        job_id = self.job_id

        model = self.model

        model_version = self.model_version

        completed_at = self.completed_at.isoformat()

        speaker_count = self.speaker_count

        validated_speakers = self.validated_speakers


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "job_id": job_id,
            "model": model,
            "model_version": model_version,
            "completed_at": completed_at,
            "speaker_count": speaker_count,
            "validated_speakers": validated_speakers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        job_id = d.pop("job_id")

        model = d.pop("model")

        model_version = d.pop("model_version")

        completed_at = datetime.datetime.fromisoformat(d.pop("completed_at"))




        speaker_count = d.pop("speaker_count")

        validated_speakers = d.pop("validated_speakers")

        voice_history_entry = cls(
            job_id=job_id,
            model=model,
            model_version=model_version,
            completed_at=completed_at,
            speaker_count=speaker_count,
            validated_speakers=validated_speakers,
        )


        voice_history_entry.additional_properties = d
        return voice_history_entry

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
