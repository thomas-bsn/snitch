from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transcription_settings_input_mode import TranscriptionSettingsInputMode
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transcription_pipeline import TranscriptionPipeline





T = TypeVar("T", bound="TranscriptionSettingsInput")



@_attrs_define
class TranscriptionSettingsInput:
    """
        Attributes:
            mode (TranscriptionSettingsInputMode):
            endpoint (str):
            model (str):
            language (str): Code de langue Whisper : l’un des 100 codes que Whisper accepte (liste fermée, codes de Whisper
                tels quels, par exemple jw pour le javanais, haw, yue). Ignoré si mode vaut disabled.
            profile (TranscriptionPipeline | Unset):
     """

    mode: TranscriptionSettingsInputMode
    endpoint: str
    model: str
    language: str
    profile: TranscriptionPipeline | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.transcription_pipeline import TranscriptionPipeline # noqa: PLC0415
        mode = self.mode.value

        endpoint = self.endpoint

        model = self.model

        language = self.language

        profile: dict[str, Any] | Unset = UNSET
        if not isinstance(self.profile, Unset):
            profile = self.profile.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "mode": mode,
            "endpoint": endpoint,
            "model": model,
            "language": language,
        })
        if profile is not UNSET:
            field_dict["profile"] = profile

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transcription_pipeline import TranscriptionPipeline # noqa: PLC0415
        d = dict(src_dict)
        mode = TranscriptionSettingsInputMode(d.pop("mode"))




        endpoint = d.pop("endpoint")

        model = d.pop("model")

        language = d.pop("language")

        _profile = d.pop("profile", UNSET)
        profile: TranscriptionPipeline | Unset
        if isinstance(_profile,  Unset):
            profile = UNSET
        else:
            profile = TranscriptionPipeline.from_dict(_profile)




        transcription_settings_input = cls(
            mode=mode,
            endpoint=endpoint,
            model=model,
            language=language,
            profile=profile,
        )


        transcription_settings_input.additional_properties = d
        return transcription_settings_input

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
