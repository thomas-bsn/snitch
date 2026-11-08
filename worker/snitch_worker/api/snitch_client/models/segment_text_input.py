from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SegmentTextInput")



@_attrs_define
class SegmentTextInput:
    """
        Attributes:
            text (str): Nouveau texte (4000 caractères au plus, non vide après suppression des espaces de bord).
            expected_version (int): Version du segment vue par l'appelant ; 0 si jamais corrigé.
            reason (str | Unset): Motif facultatif (500 caractères au plus).
            transcription_job_id (int | Unset): Transcription lue par l'appelant ; refusée (transcription_changed) si le
                clip a été retranscrit depuis.
            expected_revision_id (str | Unset): Révision lue ; 409 transcription_changed si remplacée. Facultatif pour les
                anciens clients monopistes.
     """

    text: str
    expected_version: int
    reason: str | Unset = UNSET
    transcription_job_id: int | Unset = UNSET
    expected_revision_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        text = self.text

        expected_version = self.expected_version

        reason = self.reason

        transcription_job_id = self.transcription_job_id

        expected_revision_id = self.expected_revision_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "text": text,
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
        text = d.pop("text")

        expected_version = d.pop("expected_version")

        reason = d.pop("reason", UNSET)

        transcription_job_id = d.pop("transcription_job_id", UNSET)

        expected_revision_id = d.pop("expected_revision_id", UNSET)

        segment_text_input = cls(
            text=text,
            expected_version=expected_version,
            reason=reason,
            transcription_job_id=transcription_job_id,
            expected_revision_id=expected_revision_id,
        )

        return segment_text_input
