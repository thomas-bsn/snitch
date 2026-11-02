from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.clip_visibility import ClipVisibility
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.audio_track_selection import AudioTrackSelection
  from ..models.clip_detected_type_0 import ClipDetectedType0
  from ..models.clip_origins_item import ClipOriginsItem
  from ..models.clip_person import ClipPerson
  from ..models.clip_progress import ClipProgress
  from ..models.media_track import MediaTrack
  from ..models.transcription import Transcription





T = TypeVar("T", bound="Clip")



@_attrs_define
class Clip:
    """
        Attributes:
            id (int):
            owner_id (int):
            owner (str):
            visibility (ClipVisibility):
            display_title (None | str): Titre affiché choisi par un administrateur ; absent, le nom du fichier
                (original_name) sert de titre. original_name reste le nom physique, jamais modifié.
            detected (ClipDetectedType0 | None): Valeurs détectées automatiquement, conservées à la première modification
                manuelle pour pouvoir les restaurer ; une valeur n'est restaurable que si son indicateur *_available est vrai.
            original_name (str):
            size_bytes (int):
            discovered_at (datetime.datetime):
            recorded_at (datetime.datetime | None):
            recorded_at_estimated (bool):
            format_name (str):
            video_codec (str):
            audio_codec (str):
            tracks (list[MediaTrack]):
            uploaders (list[str]):
            owner_deleted (bool | Unset): Le compte propriétaire a été supprimé.
            origins (list[ClipOriginsItem] | Unset):
            game (None | str | Unset):
            game_origin (None | str | Unset):
            capture_time_origin (None | str | Unset):
            duration_seconds (float | None | Unset):
            status (str | Unset):
            sources (list[str] | Unset):
            persons (list[ClipPerson] | Unset): Personnes identifiées dans le clip (attribution humaine ou automatique),
                triées par nom.
            transcription (Transcription | Unset):
            audio_track_selection (AudioTrackSelection | Unset):
            progress (ClipProgress | Unset): Avancement du clip pour les étapes que le serveur seul connaît. Les étapes de
                voix se lisent avec GET /api/clips/{id}/voice. Un clip est toujours importé.
     """

    id: int
    owner_id: int
    owner: str
    visibility: ClipVisibility
    display_title: None | str
    detected: ClipDetectedType0 | None
    original_name: str
    size_bytes: int
    discovered_at: datetime.datetime
    recorded_at: datetime.datetime | None
    recorded_at_estimated: bool
    format_name: str
    video_codec: str
    audio_codec: str
    tracks: list[MediaTrack]
    uploaders: list[str]
    owner_deleted: bool | Unset = UNSET
    origins: list[ClipOriginsItem] | Unset = UNSET
    game: None | str | Unset = UNSET
    game_origin: None | str | Unset = UNSET
    capture_time_origin: None | str | Unset = UNSET
    duration_seconds: float | None | Unset = UNSET
    status: str | Unset = UNSET
    sources: list[str] | Unset = UNSET
    persons: list[ClipPerson] | Unset = UNSET
    transcription: Transcription | Unset = UNSET
    audio_track_selection: AudioTrackSelection | Unset = UNSET
    progress: ClipProgress | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_track_selection import AudioTrackSelection # noqa: PLC0415
        from ..models.clip_detected_type_0 import ClipDetectedType0 # noqa: PLC0415
        from ..models.clip_origins_item import ClipOriginsItem # noqa: PLC0415
        from ..models.clip_person import ClipPerson # noqa: PLC0415
        from ..models.clip_progress import ClipProgress # noqa: PLC0415
        from ..models.media_track import MediaTrack # noqa: PLC0415
        from ..models.transcription import Transcription # noqa: PLC0415
        id = self.id

        owner_id = self.owner_id

        owner = self.owner

        visibility = self.visibility.value

        display_title: None | str
        display_title = self.display_title

        detected: dict[str, Any] | None
        if isinstance(self.detected, ClipDetectedType0):
            detected = self.detected.to_dict()
        else:
            detected = self.detected

        original_name = self.original_name

        size_bytes = self.size_bytes

        discovered_at = self.discovered_at.isoformat()

        recorded_at: None | str
        if isinstance(self.recorded_at, datetime.datetime):
            recorded_at = self.recorded_at.isoformat()
        else:
            recorded_at = self.recorded_at

        recorded_at_estimated = self.recorded_at_estimated

        format_name = self.format_name

        video_codec = self.video_codec

        audio_codec = self.audio_codec

        tracks = []
        for tracks_item_data in self.tracks:
            tracks_item = tracks_item_data.to_dict()
            tracks.append(tracks_item)



        uploaders = self.uploaders



        owner_deleted = self.owner_deleted

        origins: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.origins, Unset):
            origins = []
            for origins_item_data in self.origins:
                origins_item = origins_item_data.to_dict()
                origins.append(origins_item)



        game: None | str | Unset
        if isinstance(self.game, Unset):
            game = UNSET
        else:
            game = self.game

        game_origin: None | str | Unset
        if isinstance(self.game_origin, Unset):
            game_origin = UNSET
        else:
            game_origin = self.game_origin

        capture_time_origin: None | str | Unset
        if isinstance(self.capture_time_origin, Unset):
            capture_time_origin = UNSET
        else:
            capture_time_origin = self.capture_time_origin

        duration_seconds: float | None | Unset
        if isinstance(self.duration_seconds, Unset):
            duration_seconds = UNSET
        else:
            duration_seconds = self.duration_seconds

        status = self.status

        sources: list[str] | Unset = UNSET
        if not isinstance(self.sources, Unset):
            sources = self.sources



        persons: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.persons, Unset):
            persons = []
            for persons_item_data in self.persons:
                persons_item = persons_item_data.to_dict()
                persons.append(persons_item)



        transcription: dict[str, Any] | Unset = UNSET
        if not isinstance(self.transcription, Unset):
            transcription = self.transcription.to_dict()

        audio_track_selection: dict[str, Any] | Unset = UNSET
        if not isinstance(self.audio_track_selection, Unset):
            audio_track_selection = self.audio_track_selection.to_dict()

        progress: dict[str, Any] | Unset = UNSET
        if not isinstance(self.progress, Unset):
            progress = self.progress.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "owner_id": owner_id,
            "owner": owner,
            "visibility": visibility,
            "display_title": display_title,
            "detected": detected,
            "original_name": original_name,
            "size_bytes": size_bytes,
            "discovered_at": discovered_at,
            "recorded_at": recorded_at,
            "recorded_at_estimated": recorded_at_estimated,
            "format_name": format_name,
            "video_codec": video_codec,
            "audio_codec": audio_codec,
            "tracks": tracks,
            "uploaders": uploaders,
        })
        if owner_deleted is not UNSET:
            field_dict["owner_deleted"] = owner_deleted
        if origins is not UNSET:
            field_dict["origins"] = origins
        if game is not UNSET:
            field_dict["game"] = game
        if game_origin is not UNSET:
            field_dict["game_origin"] = game_origin
        if capture_time_origin is not UNSET:
            field_dict["capture_time_origin"] = capture_time_origin
        if duration_seconds is not UNSET:
            field_dict["duration_seconds"] = duration_seconds
        if status is not UNSET:
            field_dict["status"] = status
        if sources is not UNSET:
            field_dict["sources"] = sources
        if persons is not UNSET:
            field_dict["persons"] = persons
        if transcription is not UNSET:
            field_dict["transcription"] = transcription
        if audio_track_selection is not UNSET:
            field_dict["audio_track_selection"] = audio_track_selection
        if progress is not UNSET:
            field_dict["progress"] = progress

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_track_selection import AudioTrackSelection # noqa: PLC0415
        from ..models.clip_detected_type_0 import ClipDetectedType0 # noqa: PLC0415
        from ..models.clip_origins_item import ClipOriginsItem # noqa: PLC0415
        from ..models.clip_person import ClipPerson # noqa: PLC0415
        from ..models.clip_progress import ClipProgress # noqa: PLC0415
        from ..models.media_track import MediaTrack # noqa: PLC0415
        from ..models.transcription import Transcription # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        owner_id = d.pop("owner_id")

        owner = d.pop("owner")

        visibility = ClipVisibility(d.pop("visibility"))




        def _parse_display_title(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        display_title = _parse_display_title(d.pop("display_title"))


        def _parse_detected(data: object) -> ClipDetectedType0 | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                detected_type_0 = ClipDetectedType0.from_dict(data)



                return detected_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(ClipDetectedType0 | None, data)

        detected = _parse_detected(d.pop("detected"))


        original_name = d.pop("original_name")

        size_bytes = d.pop("size_bytes")

        discovered_at = datetime.datetime.fromisoformat(d.pop("discovered_at"))




        def _parse_recorded_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                recorded_at_type_0 = datetime.datetime.fromisoformat(data)



                return recorded_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        recorded_at = _parse_recorded_at(d.pop("recorded_at"))


        recorded_at_estimated = d.pop("recorded_at_estimated")

        format_name = d.pop("format_name")

        video_codec = d.pop("video_codec")

        audio_codec = d.pop("audio_codec")

        tracks = []
        _tracks = d.pop("tracks")
        for tracks_item_data in (_tracks):
            tracks_item = MediaTrack.from_dict(tracks_item_data)



            tracks.append(tracks_item)


        uploaders = cast(list[str], d.pop("uploaders"))


        owner_deleted = d.pop("owner_deleted", UNSET)

        _origins = d.pop("origins", UNSET)
        origins: list[ClipOriginsItem] | Unset = UNSET
        if _origins is not UNSET:
            origins = []
            for origins_item_data in _origins:
                origins_item = ClipOriginsItem.from_dict(origins_item_data)



                origins.append(origins_item)


        def _parse_game(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        game = _parse_game(d.pop("game", UNSET))


        def _parse_game_origin(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        game_origin = _parse_game_origin(d.pop("game_origin", UNSET))


        def _parse_capture_time_origin(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        capture_time_origin = _parse_capture_time_origin(d.pop("capture_time_origin", UNSET))


        def _parse_duration_seconds(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        duration_seconds = _parse_duration_seconds(d.pop("duration_seconds", UNSET))


        status = d.pop("status", UNSET)

        sources = cast(list[str], d.pop("sources", UNSET))


        _persons = d.pop("persons", UNSET)
        persons: list[ClipPerson] | Unset = UNSET
        if _persons is not UNSET:
            persons = []
            for persons_item_data in _persons:
                persons_item = ClipPerson.from_dict(persons_item_data)



                persons.append(persons_item)


        _transcription = d.pop("transcription", UNSET)
        transcription: Transcription | Unset
        if isinstance(_transcription,  Unset):
            transcription = UNSET
        else:
            transcription = Transcription.from_dict(_transcription)




        _audio_track_selection = d.pop("audio_track_selection", UNSET)
        audio_track_selection: AudioTrackSelection | Unset
        if isinstance(_audio_track_selection,  Unset):
            audio_track_selection = UNSET
        else:
            audio_track_selection = AudioTrackSelection.from_dict(_audio_track_selection)




        _progress = d.pop("progress", UNSET)
        progress: ClipProgress | Unset
        if isinstance(_progress,  Unset):
            progress = UNSET
        else:
            progress = ClipProgress.from_dict(_progress)




        clip = cls(
            id=id,
            owner_id=owner_id,
            owner=owner,
            visibility=visibility,
            display_title=display_title,
            detected=detected,
            original_name=original_name,
            size_bytes=size_bytes,
            discovered_at=discovered_at,
            recorded_at=recorded_at,
            recorded_at_estimated=recorded_at_estimated,
            format_name=format_name,
            video_codec=video_codec,
            audio_codec=audio_codec,
            tracks=tracks,
            uploaders=uploaders,
            owner_deleted=owner_deleted,
            origins=origins,
            game=game,
            game_origin=game_origin,
            capture_time_origin=capture_time_origin,
            duration_seconds=duration_seconds,
            status=status,
            sources=sources,
            persons=persons,
            transcription=transcription,
            audio_track_selection=audio_track_selection,
            progress=progress,
        )


        clip.additional_properties = d
        return clip

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
