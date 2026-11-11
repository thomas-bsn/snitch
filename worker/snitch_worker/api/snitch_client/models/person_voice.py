from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.person_voice_reference_use import PersonVoiceReferenceUse
from typing import cast
import datetime






T = TypeVar("T", bound="PersonVoice")



@_attrs_define
class PersonVoice:
    """
        Attributes:
            local_speaker_id (int):
            clip_id (int):
            clip_name (str):
            label (str):
            speaking_seconds (float):
            recorded_at (datetime.datetime): Date d'enregistrement du clip.
            sample_count (int): Échantillons vocaux actifs que cette voix fournit aux références (0 ou 1) : ce qu'une
                séparation en retirerait.
            reference_use (PersonVoiceReferenceUse): Ce que la Référence du groupe d'empreinte actif, dans le mode de score
                actif, fait de l'échantillon de cette voix : contributive (utilisé), ou la raison de son écart (en mode
                best_sample, un échantillon contributif mais non comparable est écarté comme too_short) ; null quand la voix ne
                fournit aucun échantillon actif.
     """

    local_speaker_id: int
    clip_id: int
    clip_name: str
    label: str
    speaking_seconds: float
    recorded_at: datetime.datetime
    sample_count: int
    reference_use: PersonVoiceReferenceUse
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        local_speaker_id = self.local_speaker_id

        clip_id = self.clip_id

        clip_name = self.clip_name

        label = self.label

        speaking_seconds = self.speaking_seconds

        recorded_at = self.recorded_at.isoformat()

        sample_count = self.sample_count

        reference_use = self.reference_use.value


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "local_speaker_id": local_speaker_id,
            "clip_id": clip_id,
            "clip_name": clip_name,
            "label": label,
            "speaking_seconds": speaking_seconds,
            "recorded_at": recorded_at,
            "sample_count": sample_count,
            "reference_use": reference_use,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        local_speaker_id = d.pop("local_speaker_id")

        clip_id = d.pop("clip_id")

        clip_name = d.pop("clip_name")

        label = d.pop("label")

        speaking_seconds = d.pop("speaking_seconds")

        recorded_at = datetime.datetime.fromisoformat(d.pop("recorded_at"))




        sample_count = d.pop("sample_count")

        reference_use = PersonVoiceReferenceUse(d.pop("reference_use"))




        person_voice = cls(
            local_speaker_id=local_speaker_id,
            clip_id=clip_id,
            clip_name=clip_name,
            label=label,
            speaking_seconds=speaking_seconds,
            recorded_at=recorded_at,
            sample_count=sample_count,
            reference_use=reference_use,
        )


        person_voice.additional_properties = d
        return person_voice

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
