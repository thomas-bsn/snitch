from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.job import Job
  from ..models.transcription_pipeline import TranscriptionPipeline
  from ..models.worker_job_input_tracks_item import WorkerJobInputTracksItem
  from ..models.worker_vocabulary_term import WorkerVocabularyTerm
  from ..models.worker_voice_input import WorkerVoiceInput
  from ..models.worker_voice_trial_input import WorkerVoiceTrialInput





T = TypeVar("T", bound="WorkerJobInput")



@_attrs_define
class WorkerJobInput:
    """
        Attributes:
            job (Job):
            original_name (str):
            size_bytes (int):
            video_codec (str):
            audio_codec (str):
            tracks (list[WorkerJobInputTracksItem]):
            media_url (str):
            vocabulary (list[WorkerVocabularyTerm]):
            transcription_model (str | Unset): Nom facultatif du modèle propre au Worker.
            duration_seconds (float | None | Unset):
            pipeline (TranscriptionPipeline | Unset):
            pipeline_version (str | Unset):
            voice (WorkerVoiceInput | Unset):
            voice_trial (WorkerVoiceTrialInput | Unset): Entrée d'un essai de diarisation (étapes voicetrial.*) : le modèle
                du catalogue et les options choisis pour l'essai, jamais la configuration active ni une version de pipeline.
     """

    job: Job
    original_name: str
    size_bytes: int
    video_codec: str
    audio_codec: str
    tracks: list[WorkerJobInputTracksItem]
    media_url: str
    vocabulary: list[WorkerVocabularyTerm]
    transcription_model: str | Unset = UNSET
    duration_seconds: float | None | Unset = UNSET
    pipeline: TranscriptionPipeline | Unset = UNSET
    pipeline_version: str | Unset = UNSET
    voice: WorkerVoiceInput | Unset = UNSET
    voice_trial: WorkerVoiceTrialInput | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.job import Job # noqa: PLC0415
        from ..models.transcription_pipeline import TranscriptionPipeline # noqa: PLC0415
        from ..models.worker_job_input_tracks_item import WorkerJobInputTracksItem # noqa: PLC0415
        from ..models.worker_vocabulary_term import WorkerVocabularyTerm # noqa: PLC0415
        from ..models.worker_voice_input import WorkerVoiceInput # noqa: PLC0415
        from ..models.worker_voice_trial_input import WorkerVoiceTrialInput # noqa: PLC0415
        job = self.job.to_dict()

        original_name = self.original_name

        size_bytes = self.size_bytes

        video_codec = self.video_codec

        audio_codec = self.audio_codec

        tracks = []
        for tracks_item_data in self.tracks:
            tracks_item = tracks_item_data.to_dict()
            tracks.append(tracks_item)



        media_url = self.media_url

        vocabulary = []
        for vocabulary_item_data in self.vocabulary:
            vocabulary_item = vocabulary_item_data.to_dict()
            vocabulary.append(vocabulary_item)



        transcription_model = self.transcription_model

        duration_seconds: float | None | Unset
        if isinstance(self.duration_seconds, Unset):
            duration_seconds = UNSET
        else:
            duration_seconds = self.duration_seconds

        pipeline: dict[str, Any] | Unset = UNSET
        if not isinstance(self.pipeline, Unset):
            pipeline = self.pipeline.to_dict()

        pipeline_version = self.pipeline_version

        voice: dict[str, Any] | Unset = UNSET
        if not isinstance(self.voice, Unset):
            voice = self.voice.to_dict()

        voice_trial: dict[str, Any] | Unset = UNSET
        if not isinstance(self.voice_trial, Unset):
            voice_trial = self.voice_trial.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "job": job,
            "original_name": original_name,
            "size_bytes": size_bytes,
            "video_codec": video_codec,
            "audio_codec": audio_codec,
            "tracks": tracks,
            "media_url": media_url,
            "vocabulary": vocabulary,
        })
        if transcription_model is not UNSET:
            field_dict["transcription_model"] = transcription_model
        if duration_seconds is not UNSET:
            field_dict["duration_seconds"] = duration_seconds
        if pipeline is not UNSET:
            field_dict["pipeline"] = pipeline
        if pipeline_version is not UNSET:
            field_dict["pipeline_version"] = pipeline_version
        if voice is not UNSET:
            field_dict["voice"] = voice
        if voice_trial is not UNSET:
            field_dict["voice_trial"] = voice_trial

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.job import Job # noqa: PLC0415
        from ..models.transcription_pipeline import TranscriptionPipeline # noqa: PLC0415
        from ..models.worker_job_input_tracks_item import WorkerJobInputTracksItem # noqa: PLC0415
        from ..models.worker_vocabulary_term import WorkerVocabularyTerm # noqa: PLC0415
        from ..models.worker_voice_input import WorkerVoiceInput # noqa: PLC0415
        from ..models.worker_voice_trial_input import WorkerVoiceTrialInput # noqa: PLC0415
        d = dict(src_dict)
        job = Job.from_dict(d.pop("job"))




        original_name = d.pop("original_name")

        size_bytes = d.pop("size_bytes")

        video_codec = d.pop("video_codec")

        audio_codec = d.pop("audio_codec")

        tracks = []
        _tracks = d.pop("tracks")
        for tracks_item_data in (_tracks):
            tracks_item = WorkerJobInputTracksItem.from_dict(tracks_item_data)



            tracks.append(tracks_item)


        media_url = d.pop("media_url")

        vocabulary = []
        _vocabulary = d.pop("vocabulary")
        for vocabulary_item_data in (_vocabulary):
            vocabulary_item = WorkerVocabularyTerm.from_dict(vocabulary_item_data)



            vocabulary.append(vocabulary_item)


        transcription_model = d.pop("transcription_model", UNSET)

        def _parse_duration_seconds(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        duration_seconds = _parse_duration_seconds(d.pop("duration_seconds", UNSET))


        _pipeline = d.pop("pipeline", UNSET)
        pipeline: TranscriptionPipeline | Unset
        if isinstance(_pipeline,  Unset):
            pipeline = UNSET
        else:
            pipeline = TranscriptionPipeline.from_dict(_pipeline)




        pipeline_version = d.pop("pipeline_version", UNSET)

        _voice = d.pop("voice", UNSET)
        voice: WorkerVoiceInput | Unset
        if isinstance(_voice,  Unset):
            voice = UNSET
        else:
            voice = WorkerVoiceInput.from_dict(_voice)




        _voice_trial = d.pop("voice_trial", UNSET)
        voice_trial: WorkerVoiceTrialInput | Unset
        if isinstance(_voice_trial,  Unset):
            voice_trial = UNSET
        else:
            voice_trial = WorkerVoiceTrialInput.from_dict(_voice_trial)




        worker_job_input = cls(
            job=job,
            original_name=original_name,
            size_bytes=size_bytes,
            video_codec=video_codec,
            audio_codec=audio_codec,
            tracks=tracks,
            media_url=media_url,
            vocabulary=vocabulary,
            transcription_model=transcription_model,
            duration_seconds=duration_seconds,
            pipeline=pipeline,
            pipeline_version=pipeline_version,
            voice=voice,
            voice_trial=voice_trial,
        )


        worker_job_input.additional_properties = d
        return worker_job_input

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
