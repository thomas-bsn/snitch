from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transcript_segment import TranscriptSegment
  from ..models.transcription_local_speakers_item import TranscriptionLocalSpeakersItem





T = TypeVar("T", bound="Transcription")



@_attrs_define
class Transcription:
    """
        Attributes:
            provider (str):
            model (str):
            audio_stream_index (int):
            segments (list[TranscriptSegment]):
            version (str | Unset):
            language (str | Unset):
            transcription_job_id (int | Unset): Tâche de transcription visible : avec l'index, l'identité d'un segment.
            local_speakers (list[TranscriptionLocalSpeakersItem] | Unset): Locuteurs locaux de la piste audio, désignables
                pour un segment ; absent sans diarisation.
     """

    provider: str
    model: str
    audio_stream_index: int
    segments: list[TranscriptSegment]
    version: str | Unset = UNSET
    language: str | Unset = UNSET
    transcription_job_id: int | Unset = UNSET
    local_speakers: list[TranscriptionLocalSpeakersItem] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.transcript_segment import TranscriptSegment # noqa: PLC0415
        from ..models.transcription_local_speakers_item import TranscriptionLocalSpeakersItem # noqa: PLC0415
        provider = self.provider

        model = self.model

        audio_stream_index = self.audio_stream_index

        segments = []
        for segments_item_data in self.segments:
            segments_item = segments_item_data.to_dict()
            segments.append(segments_item)



        version = self.version

        language = self.language

        transcription_job_id = self.transcription_job_id

        local_speakers: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.local_speakers, Unset):
            local_speakers = []
            for local_speakers_item_data in self.local_speakers:
                local_speakers_item = local_speakers_item_data.to_dict()
                local_speakers.append(local_speakers_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "provider": provider,
            "model": model,
            "audio_stream_index": audio_stream_index,
            "segments": segments,
        })
        if version is not UNSET:
            field_dict["version"] = version
        if language is not UNSET:
            field_dict["language"] = language
        if transcription_job_id is not UNSET:
            field_dict["transcription_job_id"] = transcription_job_id
        if local_speakers is not UNSET:
            field_dict["local_speakers"] = local_speakers

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transcript_segment import TranscriptSegment # noqa: PLC0415
        from ..models.transcription_local_speakers_item import TranscriptionLocalSpeakersItem # noqa: PLC0415
        d = dict(src_dict)
        provider = d.pop("provider")

        model = d.pop("model")

        audio_stream_index = d.pop("audio_stream_index")

        segments = []
        _segments = d.pop("segments")
        for segments_item_data in (_segments):
            segments_item = TranscriptSegment.from_dict(segments_item_data)



            segments.append(segments_item)


        version = d.pop("version", UNSET)

        language = d.pop("language", UNSET)

        transcription_job_id = d.pop("transcription_job_id", UNSET)

        _local_speakers = d.pop("local_speakers", UNSET)
        local_speakers: list[TranscriptionLocalSpeakersItem] | Unset = UNSET
        if _local_speakers is not UNSET:
            local_speakers = []
            for local_speakers_item_data in _local_speakers:
                local_speakers_item = TranscriptionLocalSpeakersItem.from_dict(local_speakers_item_data)



                local_speakers.append(local_speakers_item)


        transcription = cls(
            provider=provider,
            model=model,
            audio_stream_index=audio_stream_index,
            segments=segments,
            version=version,
            language=language,
            transcription_job_id=transcription_job_id,
            local_speakers=local_speakers,
        )


        transcription.additional_properties = d
        return transcription

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
