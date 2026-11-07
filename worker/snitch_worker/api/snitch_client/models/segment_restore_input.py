from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SegmentRestoreInput")



@_attrs_define
class SegmentRestoreInput:
    """
        Attributes:
            expected_version (int): Version du segment vue par l'appelant.
            original (bool | Unset): Vrai pour restaurer le texte d'origine du worker ; exclusif avec event_id.
            event_id (int | Unset): Entrée d'historique de ce segment dont le nouveau texte est restauré ; exclusif avec
                original.
            reason (str | Unset): Motif facultatif (500 caractères au plus).
            transcription_job_id (int | Unset):
            expected_revision_id (str | Unset): Révision lue ; 409 transcription_changed si remplacée. Facultatif pour les
                anciens clients monopistes.
     """

    expected_version: int
    original: bool | Unset = UNSET
    event_id: int | Unset = UNSET
    reason: str | Unset = UNSET
    transcription_job_id: int | Unset = UNSET
    expected_revision_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        expected_version = self.expected_version

        original = self.original

        event_id = self.event_id

        reason = self.reason

        transcription_job_id = self.transcription_job_id

        expected_revision_id = self.expected_revision_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "expected_version": expected_version,
        })
        if original is not UNSET:
            field_dict["original"] = original
        if event_id is not UNSET:
            field_dict["event_id"] = event_id
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
        expected_version = d.pop("expected_version")

        original = d.pop("original", UNSET)

        event_id = d.pop("event_id", UNSET)

        reason = d.pop("reason", UNSET)

        transcription_job_id = d.pop("transcription_job_id", UNSET)

        expected_revision_id = d.pop("expected_revision_id", UNSET)

        segment_restore_input = cls(
            expected_version=expected_version,
            original=original,
            event_id=event_id,
            reason=reason,
            transcription_job_id=transcription_job_id,
            expected_revision_id=expected_revision_id,
        )

        return segment_restore_input
