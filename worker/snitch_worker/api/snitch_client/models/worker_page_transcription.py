from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_page_transcription_mode import WorkerPageTranscriptionMode






T = TypeVar("T", bound="WorkerPageTranscription")



@_attrs_define
class WorkerPageTranscription:
    """ Profil de transcription actif de la bibliothèque.

        Attributes:
            mode (WorkerPageTranscriptionMode):
            endpoint (str):
            settings_version (str):
            model (str):
            language (str):
            version (str):
     """

    mode: WorkerPageTranscriptionMode
    endpoint: str
    settings_version: str
    model: str
    language: str
    version: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value

        endpoint = self.endpoint

        settings_version = self.settings_version

        model = self.model

        language = self.language

        version = self.version


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "mode": mode,
            "endpoint": endpoint,
            "settings_version": settings_version,
            "model": model,
            "language": language,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = WorkerPageTranscriptionMode(d.pop("mode"))




        endpoint = d.pop("endpoint")

        settings_version = d.pop("settings_version")

        model = d.pop("model")

        language = d.pop("language")

        version = d.pop("version")

        worker_page_transcription = cls(
            mode=mode,
            endpoint=endpoint,
            settings_version=settings_version,
            model=model,
            language=language,
            version=version,
        )


        worker_page_transcription.additional_properties = d
        return worker_page_transcription

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
