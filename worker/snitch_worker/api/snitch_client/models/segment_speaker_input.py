from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="SegmentSpeakerInput")



@_attrs_define
class SegmentSpeakerInput:
    """
        Attributes:
            local_speaker_id (int | None): Locuteur local du même clip et de la même piste audio ; null pour revenir au
                locuteur déduit des tours de parole.
            expected_version (int): Version du segment vue par l'appelant ; 0 si jamais écrit.
            reason (str | Unset): Motif facultatif (500 caractères au plus).
            transcription_job_id (int | Unset): Transcription lue par l'appelant ; refusée (transcription_changed) si le
                clip a été retranscrit depuis.
            expected_revision_id (str | Unset): Révision lue ; 409 transcription_changed si remplacée. Facultatif pour les
                anciens clients monopistes.
     """

    local_speaker_id: int | None
    expected_version: int
    reason: str | Unset = UNSET
    transcription_job_id: int | Unset = UNSET
    expected_revision_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        local_speaker_id: int | None
        local_speaker_id = self.local_speaker_id

        expected_version = self.expected_version

        reason = self.reason

        transcription_job_id = self.transcription_job_id

        expected_revision_id = self.expected_revision_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "local_speaker_id": local_speaker_id,
            "expected_version": expected_version,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason
        if transcription_job_id is not UNSET:
            field_dict["transcription_job_id"] = transcription_job_id
        if expected_revision_id is not UNSET:
            field_dict["expected_revision_id"] = expected_revision_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        def _parse_local_speaker_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        local_speaker_id = _parse_local_speaker_id(d.pop("local_speaker_id"))


        expected_version = d.pop("expected_version")

        reason = d.pop("reason", UNSET)

        transcription_job_id = d.pop("transcription_job_id", UNSET)

        expected_revision_id = d.pop("expected_revision_id", UNSET)

        segment_speaker_input = cls(
            local_speaker_id=local_speaker_id,
            expected_version=expected_version,
            reason=reason,
            transcription_job_id=transcription_job_id,
            expected_revision_id=expected_revision_id,
        )

        return segment_speaker_input
