from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audio_plan_voice_status import AudioPlanVoiceStatus
from typing import cast

if TYPE_CHECKING:
  from ..models.audio_plan_voice_stake import AudioPlanVoiceStake
  from ..models.audio_plan_voice_track import AudioPlanVoiceTrack





T = TypeVar("T", bound="AudioPlanVoice")



@_attrs_define
class AudioPlanVoice:
    """ Analyse des voix de toutes les pistes retenues d'un Plan publié. La transcription publiée n'est jamais retirée par
    une panne ou l'absence d'un Worker vocal.

        Attributes:
            plan_id (int):
            clip_id (int):
            revision_id (str):
            published (bool): La révision du Plan est la transcription visible. Les étapes vocales ne démarrent qu'alors.
            status (AudioPlanVoiceStatus):
            cause (str):
            pipeline_version (str): Version du pipeline vocal épinglée par le Plan.
            tracks (list[AudioPlanVoiceTrack]):
            stake (AudioPlanVoiceStake): Travail humain qu'une diarisation des pistes concernées rendrait obsolète. Les
                anciennes voix restent protégées jusqu'à un nouveau résultat réussi.
     """

    plan_id: int
    clip_id: int
    revision_id: str
    published: bool
    status: AudioPlanVoiceStatus
    cause: str
    pipeline_version: str
    tracks: list[AudioPlanVoiceTrack]
    stake: AudioPlanVoiceStake
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_plan_voice_stake import AudioPlanVoiceStake # noqa: PLC0415
        from ..models.audio_plan_voice_track import AudioPlanVoiceTrack # noqa: PLC0415
        plan_id = self.plan_id

        clip_id = self.clip_id

        revision_id = self.revision_id

        published = self.published

        status = self.status.value

        cause = self.cause

        pipeline_version = self.pipeline_version

        tracks = []
        for tracks_item_data in self.tracks:
            tracks_item = tracks_item_data.to_dict()
            tracks.append(tracks_item)



        stake = self.stake.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "plan_id": plan_id,
            "clip_id": clip_id,
            "revision_id": revision_id,
            "published": published,
            "status": status,
            "cause": cause,
            "pipeline_version": pipeline_version,
            "tracks": tracks,
            "stake": stake,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_plan_voice_stake import AudioPlanVoiceStake # noqa: PLC0415
        from ..models.audio_plan_voice_track import AudioPlanVoiceTrack # noqa: PLC0415
        d = dict(src_dict)
        plan_id = d.pop("plan_id")

        clip_id = d.pop("clip_id")

        revision_id = d.pop("revision_id")

        published = d.pop("published")

        status = AudioPlanVoiceStatus(d.pop("status"))




        cause = d.pop("cause")

        pipeline_version = d.pop("pipeline_version")

        tracks = []
        _tracks = d.pop("tracks")
        for tracks_item_data in (_tracks):
            tracks_item = AudioPlanVoiceTrack.from_dict(tracks_item_data)



            tracks.append(tracks_item)


        stake = AudioPlanVoiceStake.from_dict(d.pop("stake"))




        audio_plan_voice = cls(
            plan_id=plan_id,
            clip_id=clip_id,
            revision_id=revision_id,
            published=published,
            status=status,
            cause=cause,
            pipeline_version=pipeline_version,
            tracks=tracks,
            stake=stake,
        )


        audio_plan_voice.additional_properties = d
        return audio_plan_voice

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
