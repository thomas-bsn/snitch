from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_transcription_config_error_code import WorkerTranscriptionConfigErrorCode
from ..types import UNSET, Unset






T = TypeVar("T", bound="WorkerTranscriptionConfig")



@_attrs_define
class WorkerTranscriptionConfig:
    """ Statut de synchronisation de la configuration de transcription pour un Worker.

        Attributes:
            received_version (str | Unset): Version de configuration reçue par le Worker.
            tested_version (str | Unset): Version de configuration testée par le Worker.
            active_version (str | Unset): Version de configuration active (validée) pour ce Worker.
            error_code (WorkerTranscriptionConfigErrorCode | Unset): Code d'erreur du dernier test.
            endpoint_override (str | Unset): Adresse Whisper propre à ce Worker, si définie.
     """

    received_version: str | Unset = UNSET
    tested_version: str | Unset = UNSET
    active_version: str | Unset = UNSET
    error_code: WorkerTranscriptionConfigErrorCode | Unset = UNSET
    endpoint_override: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        received_version = self.received_version

        tested_version = self.tested_version

        active_version = self.active_version

        error_code: str | Unset = UNSET
        if not isinstance(self.error_code, Unset):
            error_code = self.error_code.value


        endpoint_override = self.endpoint_override


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if received_version is not UNSET:
            field_dict["received_version"] = received_version
        if tested_version is not UNSET:
            field_dict["tested_version"] = tested_version
        if active_version is not UNSET:
            field_dict["active_version"] = active_version
        if error_code is not UNSET:
            field_dict["error_code"] = error_code
        if endpoint_override is not UNSET:
            field_dict["endpoint_override"] = endpoint_override

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        received_version = d.pop("received_version", UNSET)

        tested_version = d.pop("tested_version", UNSET)

        active_version = d.pop("active_version", UNSET)

        _error_code = d.pop("error_code", UNSET)
        error_code: WorkerTranscriptionConfigErrorCode | Unset
        if isinstance(_error_code,  Unset):
            error_code = UNSET
        else:
            error_code = WorkerTranscriptionConfigErrorCode(_error_code)




        endpoint_override = d.pop("endpoint_override", UNSET)

        worker_transcription_config = cls(
            received_version=received_version,
            tested_version=tested_version,
            active_version=active_version,
            error_code=error_code,
            endpoint_override=endpoint_override,
        )

        return worker_transcription_config
