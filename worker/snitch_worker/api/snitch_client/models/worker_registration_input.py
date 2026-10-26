from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.worker_transcription_profile import WorkerTranscriptionProfile





T = TypeVar("T", bound="WorkerRegistrationInput")



@_attrs_define
class WorkerRegistrationInput:
    """
        Attributes:
            name (str):
            max_concurrency (int):
            transcription_url (str | Unset): Adresse du serveur Whisper de ce worker (http ou https, sans identifiants).
                Écrite dans le fichier d'environnement généré ; non conservée.
            transcription (WorkerTranscriptionProfile | Unset):
     """

    name: str
    max_concurrency: int
    transcription_url: str | Unset = UNSET
    transcription: WorkerTranscriptionProfile | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_transcription_profile import WorkerTranscriptionProfile # noqa: PLC0415
        name = self.name

        max_concurrency = self.max_concurrency

        transcription_url = self.transcription_url

        transcription: dict[str, Any] | Unset = UNSET
        if not isinstance(self.transcription, Unset):
            transcription = self.transcription.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "max_concurrency": max_concurrency,
        })
        if transcription_url is not UNSET:
            field_dict["transcription_url"] = transcription_url
        if transcription is not UNSET:
            field_dict["transcription"] = transcription

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_transcription_profile import WorkerTranscriptionProfile # noqa: PLC0415
        d = dict(src_dict)
        name = d.pop("name")

        max_concurrency = d.pop("max_concurrency")

        transcription_url = d.pop("transcription_url", UNSET)

        _transcription = d.pop("transcription", UNSET)
        transcription: WorkerTranscriptionProfile | Unset
        if isinstance(_transcription,  Unset):
            transcription = UNSET
        else:
            transcription = WorkerTranscriptionProfile.from_dict(_transcription)




        worker_registration_input = cls(
            name=name,
            max_concurrency=max_concurrency,
            transcription_url=transcription_url,
            transcription=transcription,
        )

        return worker_registration_input
