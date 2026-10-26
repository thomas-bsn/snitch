from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.clip_microphone_track_selection_source import ClipMicrophoneTrackSelectionSource
from ..models.microphone_track_status import MicrophoneTrackStatus
from typing import cast

if TYPE_CHECKING:
  from ..models.microphone_track_event import MicrophoneTrackEvent
  from ..models.microphone_track_item import MicrophoneTrackItem
  from ..models.person_ref import PersonRef





T = TypeVar("T", bound="ClipMicrophoneTrack")



@_attrs_define
class ClipMicrophoneTrack:
    """
        Attributes:
            clip_id (int):
            status (MicrophoneTrackStatus): none : aucune règle ; designated : piste micro résolue ; disabled : le clip n'a
                pas de piste micro ; track_not_found, ambiguous_track, conflicting_sources : la désignation ne correspond pas à
                une piste unique et n'est pas appliquée ; blocked_by_validation : la piste désignée porte des voix déjà validées
                par un humain ; owner_profile_deleted : le propriétaire désigné a supprimé son profil vocal, rien ne lui est
                attribué.
            selection_source (ClipMicrophoneTrackSelectionSource):
            audio_stream_index (int | None):
            person (None | PersonRef):
            candidates (list[int]): Pistes qui correspondent toutes à la règle (ambiguous_track).
            local_speaker_id (int | None): Locuteur local de la piste micro, une fois la désignation appliquée.
            applied (bool):
            tracks (list[MicrophoneTrackItem]):
            history (list[MicrophoneTrackEvent]): Changements de règle de la source et remplacements du clip, du plus récent
                au plus ancien (50 au plus).
     """

    clip_id: int
    status: MicrophoneTrackStatus
    selection_source: ClipMicrophoneTrackSelectionSource
    audio_stream_index: int | None
    person: None | PersonRef
    candidates: list[int]
    local_speaker_id: int | None
    applied: bool
    tracks: list[MicrophoneTrackItem]
    history: list[MicrophoneTrackEvent]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.microphone_track_event import MicrophoneTrackEvent # noqa: PLC0415
        from ..models.microphone_track_item import MicrophoneTrackItem # noqa: PLC0415
        from ..models.person_ref import PersonRef # noqa: PLC0415
        clip_id = self.clip_id

        status = self.status.value

        selection_source = self.selection_source.value

        audio_stream_index: int | None
        audio_stream_index = self.audio_stream_index

        person: dict[str, Any] | None
        if isinstance(self.person, PersonRef):
            person = self.person.to_dict()
        else:
            person = self.person

        candidates = self.candidates



        local_speaker_id: int | None
        local_speaker_id = self.local_speaker_id

        applied = self.applied

        tracks = []
        for tracks_item_data in self.tracks:
            tracks_item = tracks_item_data.to_dict()
            tracks.append(tracks_item)



        history = []
        for history_item_data in self.history:
            history_item = history_item_data.to_dict()
            history.append(history_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "clip_id": clip_id,
            "status": status,
            "selection_source": selection_source,
            "audio_stream_index": audio_stream_index,
            "person": person,
            "candidates": candidates,
            "local_speaker_id": local_speaker_id,
            "applied": applied,
            "tracks": tracks,
            "history": history,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.microphone_track_event import MicrophoneTrackEvent # noqa: PLC0415
        from ..models.microphone_track_item import MicrophoneTrackItem # noqa: PLC0415
        from ..models.person_ref import PersonRef # noqa: PLC0415
        d = dict(src_dict)
        clip_id = d.pop("clip_id")

        status = MicrophoneTrackStatus(d.pop("status"))




        selection_source = ClipMicrophoneTrackSelectionSource(d.pop("selection_source"))




        def _parse_audio_stream_index(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        audio_stream_index = _parse_audio_stream_index(d.pop("audio_stream_index"))


        def _parse_person(data: object) -> None | PersonRef:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                person_type_1 = PersonRef.from_dict(data)



                return person_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | PersonRef, data)

        person = _parse_person(d.pop("person"))


        candidates = cast(list[int], d.pop("candidates"))


        def _parse_local_speaker_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        local_speaker_id = _parse_local_speaker_id(d.pop("local_speaker_id"))


        applied = d.pop("applied")

        tracks = []
        _tracks = d.pop("tracks")
        for tracks_item_data in (_tracks):
            tracks_item = MicrophoneTrackItem.from_dict(tracks_item_data)



            tracks.append(tracks_item)


        history = []
        _history = d.pop("history")
        for history_item_data in (_history):
            history_item = MicrophoneTrackEvent.from_dict(history_item_data)



            history.append(history_item)


        clip_microphone_track = cls(
            clip_id=clip_id,
            status=status,
            selection_source=selection_source,
            audio_stream_index=audio_stream_index,
            person=person,
            candidates=candidates,
            local_speaker_id=local_speaker_id,
            applied=applied,
            tracks=tracks,
            history=history,
        )


        clip_microphone_track.additional_properties = d
        return clip_microphone_track

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
