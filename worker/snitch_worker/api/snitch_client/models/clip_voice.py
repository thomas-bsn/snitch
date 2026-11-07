from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.clip_voice_status import ClipVoiceStatus
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.audio_plan_voice_track import AudioPlanVoiceTrack
  from ..models.local_speaker import LocalSpeaker
  from ..models.voice_history_entry import VoiceHistoryEntry
  from ..models.voice_step import VoiceStep





T = TypeVar("T", bound="ClipVoice")



@_attrs_define
class ClipVoice:
    """
        Attributes:
            clip_id (int):
            status (ClipVoiceStatus):
            steps (list[VoiceStep]):
            speakers (list[LocalSpeaker]):
            history (list[VoiceHistoryEntry]): Résultats de diarisation remplacés par une relance, du plus récent au plus
                ancien ; le résultat visible n'est remplacé qu'après une relance réussie.
            plan_id (int | Unset): Plan audio dont la révision est la transcription visible ; présent seulement dans ce cas.
                Les étapes et les voix couvrent alors toutes ses pistes retenues.
            tracks (list[AudioPlanVoiceTrack] | Unset): État vocal de chaque piste retenue du Plan (présent avec plan_id).
            audio_stream_index (int | None | Unset):
     """

    clip_id: int
    status: ClipVoiceStatus
    steps: list[VoiceStep]
    speakers: list[LocalSpeaker]
    history: list[VoiceHistoryEntry]
    plan_id: int | Unset = UNSET
    tracks: list[AudioPlanVoiceTrack] | Unset = UNSET
    audio_stream_index: int | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_plan_voice_track import AudioPlanVoiceTrack # noqa: PLC0415
        from ..models.local_speaker import LocalSpeaker # noqa: PLC0415
        from ..models.voice_history_entry import VoiceHistoryEntry # noqa: PLC0415
        from ..models.voice_step import VoiceStep # noqa: PLC0415
        clip_id = self.clip_id

        status = self.status.value

        steps = []
        for steps_item_data in self.steps:
            steps_item = steps_item_data.to_dict()
            steps.append(steps_item)



        speakers = []
        for speakers_item_data in self.speakers:
            speakers_item = speakers_item_data.to_dict()
            speakers.append(speakers_item)



        history = []
        for history_item_data in self.history:
            history_item = history_item_data.to_dict()
            history.append(history_item)



        plan_id = self.plan_id

        tracks: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.tracks, Unset):
            tracks = []
            for tracks_item_data in self.tracks:
                tracks_item = tracks_item_data.to_dict()
                tracks.append(tracks_item)



        audio_stream_index: int | None | Unset
        if isinstance(self.audio_stream_index, Unset):
            audio_stream_index = UNSET
        else:
            audio_stream_index = self.audio_stream_index


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "clip_id": clip_id,
            "status": status,
            "steps": steps,
            "speakers": speakers,
            "history": history,
        })
        if plan_id is not UNSET:
            field_dict["plan_id"] = plan_id
        if tracks is not UNSET:
            field_dict["tracks"] = tracks
        if audio_stream_index is not UNSET:
            field_dict["audio_stream_index"] = audio_stream_index

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_plan_voice_track import AudioPlanVoiceTrack # noqa: PLC0415
        from ..models.local_speaker import LocalSpeaker # noqa: PLC0415
        from ..models.voice_history_entry import VoiceHistoryEntry # noqa: PLC0415
        from ..models.voice_step import VoiceStep # noqa: PLC0415
        d = dict(src_dict)
        clip_id = d.pop("clip_id")

        status = ClipVoiceStatus(d.pop("status"))




        steps = []
        _steps = d.pop("steps")
        for steps_item_data in (_steps):
            steps_item = VoiceStep.from_dict(steps_item_data)



            steps.append(steps_item)


        speakers = []
        _speakers = d.pop("speakers")
        for speakers_item_data in (_speakers):
            speakers_item = LocalSpeaker.from_dict(speakers_item_data)



            speakers.append(speakers_item)


        history = []
        _history = d.pop("history")
        for history_item_data in (_history):
            history_item = VoiceHistoryEntry.from_dict(history_item_data)



            history.append(history_item)


        plan_id = d.pop("plan_id", UNSET)

        _tracks = d.pop("tracks", UNSET)
        tracks: list[AudioPlanVoiceTrack] | Unset = UNSET
        if _tracks is not UNSET:
            tracks = []
            for tracks_item_data in _tracks:
                tracks_item = AudioPlanVoiceTrack.from_dict(tracks_item_data)



                tracks.append(tracks_item)


        def _parse_audio_stream_index(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        audio_stream_index = _parse_audio_stream_index(d.pop("audio_stream_index", UNSET))


        clip_voice = cls(
            clip_id=clip_id,
            status=status,
            steps=steps,
            speakers=speakers,
            history=history,
            plan_id=plan_id,
            tracks=tracks,
            audio_stream_index=audio_stream_index,
        )


        clip_voice.additional_properties = d
        return clip_voice

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
