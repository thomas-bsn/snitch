from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="Person")



@_attrs_define
class Person:
    """
        Attributes:
            id (int):
            display_name (str):
            user_id (int | None): Compte lié, le cas échéant : une Personne n'est jamais un compte.
            validated_sample_count (int): Échantillons vocaux validés actifs, donc références de la Personne. Ce nombre
                n'est pas une garantie de qualité d'identification.
            local_speaker_count (int): Locuteurs locaux validés pour cette Personne.
            clip_count (int): Vidéos où une voix validée de la Personne est entendue, parmi celles que l'appelant peut voir.
                Rempli en lecture (liste et fiche), 0 dans le résultat d'une correction.
            speaking_seconds (float): Temps de parole cumulé de ces voix validées, sur les vidéos visibles par l'appelant.
            voice_profile_deleted_at (datetime.datetime | None): Date de suppression du profil vocal : tant qu'elle est
                renseignée, le rapprochement ignore les références de la Personne et aucune validation vers elle n'est acceptée.
                Aucune donnée vocale n'est conservée.
            created_at (datetime.datetime):
     """

    id: int
    display_name: str
    user_id: int | None
    validated_sample_count: int
    local_speaker_count: int
    clip_count: int
    speaking_seconds: float
    voice_profile_deleted_at: datetime.datetime | None
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        display_name = self.display_name

        user_id: int | None
        user_id = self.user_id

        validated_sample_count = self.validated_sample_count

        local_speaker_count = self.local_speaker_count

        clip_count = self.clip_count

        speaking_seconds = self.speaking_seconds

        voice_profile_deleted_at: None | str
        if isinstance(self.voice_profile_deleted_at, datetime.datetime):
            voice_profile_deleted_at = self.voice_profile_deleted_at.isoformat()
        else:
            voice_profile_deleted_at = self.voice_profile_deleted_at

        created_at = self.created_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "display_name": display_name,
            "user_id": user_id,
            "validated_sample_count": validated_sample_count,
            "local_speaker_count": local_speaker_count,
            "clip_count": clip_count,
            "speaking_seconds": speaking_seconds,
            "voice_profile_deleted_at": voice_profile_deleted_at,
            "created_at": created_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        display_name = d.pop("display_name")

        def _parse_user_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        user_id = _parse_user_id(d.pop("user_id"))


        validated_sample_count = d.pop("validated_sample_count")

        local_speaker_count = d.pop("local_speaker_count")

        clip_count = d.pop("clip_count")

        speaking_seconds = d.pop("speaking_seconds")

        def _parse_voice_profile_deleted_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                voice_profile_deleted_at_type_0 = datetime.datetime.fromisoformat(data)



                return voice_profile_deleted_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        voice_profile_deleted_at = _parse_voice_profile_deleted_at(d.pop("voice_profile_deleted_at"))


        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        person = cls(
            id=id,
            display_name=display_name,
            user_id=user_id,
            validated_sample_count=validated_sample_count,
            local_speaker_count=local_speaker_count,
            clip_count=clip_count,
            speaking_seconds=speaking_seconds,
            voice_profile_deleted_at=voice_profile_deleted_at,
            created_at=created_at,
        )


        person.additional_properties = d
        return person

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
