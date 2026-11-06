from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.speaker_coverage import SpeakerCoverage
  from ..models.voice_trace_diarization_type_0 import VoiceTraceDiarizationType0
  from ..models.voice_trace_segments_item import VoiceTraceSegmentsItem
  from ..models.voice_trace_transcription import VoiceTraceTranscription
  from ..models.voice_trace_turns_item import VoiceTraceTurnsItem





T = TypeVar("T", bound="VoiceTrace")



@_attrs_define
class VoiceTrace:
    """ Trace de l'alignement diarisation / transcription de la piste visible (#273), sans aucun texte transcrit.

        Attributes:
            clip_id (int):
            audio_stream_index (int):
            transcription (VoiceTraceTranscription):
            diarization (None | VoiceTraceDiarizationType0): Null sans diarisation réussie pour cette piste.
            turns (list[VoiceTraceTurnsItem]): Tours bruts de la diarisation, voix ignorées exclues.
            segments (list[VoiceTraceSegmentsItem]): Segments bruts de la transcription avec les voix qui les recouvrent
                (avant alignement) et la voix retenue (après).
            coverage (list[SpeakerCoverage]):
     """

    clip_id: int
    audio_stream_index: int
    transcription: VoiceTraceTranscription
    diarization: None | VoiceTraceDiarizationType0
    turns: list[VoiceTraceTurnsItem]
    segments: list[VoiceTraceSegmentsItem]
    coverage: list[SpeakerCoverage]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.speaker_coverage import SpeakerCoverage # noqa: PLC0415
        from ..models.voice_trace_diarization_type_0 import VoiceTraceDiarizationType0 # noqa: PLC0415
        from ..models.voice_trace_segments_item import VoiceTraceSegmentsItem # noqa: PLC0415
        from ..models.voice_trace_transcription import VoiceTraceTranscription # noqa: PLC0415
        from ..models.voice_trace_turns_item import VoiceTraceTurnsItem # noqa: PLC0415
        clip_id = self.clip_id

        audio_stream_index = self.audio_stream_index

        transcription = self.transcription.to_dict()

        diarization: dict[str, Any] | None
        if isinstance(self.diarization, VoiceTraceDiarizationType0):
            diarization = self.diarization.to_dict()
        else:
            diarization = self.diarization

        turns = []
        for turns_item_data in self.turns:
            turns_item = turns_item_data.to_dict()
            turns.append(turns_item)



        segments = []
        for segments_item_data in self.segments:
            segments_item = segments_item_data.to_dict()
            segments.append(segments_item)



        coverage = []
        for coverage_item_data in self.coverage:
            coverage_item = coverage_item_data.to_dict()
            coverage.append(coverage_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "clip_id": clip_id,
            "audio_stream_index": audio_stream_index,
            "transcription": transcription,
            "diarization": diarization,
            "turns": turns,
            "segments": segments,
            "coverage": coverage,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.speaker_coverage import SpeakerCoverage # noqa: PLC0415
        from ..models.voice_trace_diarization_type_0 import VoiceTraceDiarizationType0 # noqa: PLC0415
        from ..models.voice_trace_segments_item import VoiceTraceSegmentsItem # noqa: PLC0415
        from ..models.voice_trace_transcription import VoiceTraceTranscription # noqa: PLC0415
        from ..models.voice_trace_turns_item import VoiceTraceTurnsItem # noqa: PLC0415
        d = dict(src_dict)
        clip_id = d.pop("clip_id")

        audio_stream_index = d.pop("audio_stream_index")

        transcription = VoiceTraceTranscription.from_dict(d.pop("transcription"))




        def _parse_diarization(data: object) -> None | VoiceTraceDiarizationType0:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                diarization_type_0 = VoiceTraceDiarizationType0.from_dict(data)



                return diarization_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | VoiceTraceDiarizationType0, data)

        diarization = _parse_diarization(d.pop("diarization"))


        turns = []
        _turns = d.pop("turns")
        for turns_item_data in (_turns):
            turns_item = VoiceTraceTurnsItem.from_dict(turns_item_data)



            turns.append(turns_item)


        segments = []
        _segments = d.pop("segments")
        for segments_item_data in (_segments):
            segments_item = VoiceTraceSegmentsItem.from_dict(segments_item_data)



            segments.append(segments_item)


        coverage = []
        _coverage = d.pop("coverage")
        for coverage_item_data in (_coverage):
            coverage_item = SpeakerCoverage.from_dict(coverage_item_data)



            coverage.append(coverage_item)


        voice_trace = cls(
            clip_id=clip_id,
            audio_stream_index=audio_stream_index,
            transcription=transcription,
            diarization=diarization,
            turns=turns,
            segments=segments,
            coverage=coverage,
        )


        voice_trace.additional_properties = d
        return voice_trace

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
