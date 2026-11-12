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
  from ..models.worker_audio_dedup_input import WorkerAudioDedupInput
  from ..models.worker_audio_diagnostic_input import WorkerAudioDiagnosticInput
  from ..models.worker_audio_plan_task import WorkerAudioPlanTask
  from ..models.worker_job_input_tracks_item import WorkerJobInputTracksItem
  from ..models.worker_separation_input import WorkerSeparationInput
  from ..models.worker_vocabulary_term import WorkerVocabularyTerm
  from ..models.worker_voice_input import WorkerVoiceInput





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
            audio_plan (WorkerAudioPlanTask | Unset): Tâche d’un job de transcription épinglé à un plan audio (capacité
                audio.multitrack.v1). Les intervalles sont relatifs à la piste ; le Worker ajoute start_seconds une seule fois
                et rend des temps sur la ligne de temps du clip.
            audio_diagnostic (WorkerAudioDiagnosticInput | Unset): Identité immuable d'un job audio.diagnostic.N.
                start_seconds et duration_seconds viennent de l'inventaire du plan (flux ffprobe), jamais du conteneur.
            audio_dedup (WorkerAudioDedupInput | Unset): Tâche d'un job audio.dedup.N (comparaison acoustique des pistes
                retenues d'un plan). Les segments sont dans le repère du clip ; start_seconds est le décalage du flux de la
                piste.
            separation (WorkerSeparationInput | Unset): Tâche benchmark.separation.N (#402) : le modèle de séparation
                demandé et la transcription de la chaîne, dans le repère de la piste. Les tours sont dans voice.diarization, le
                modèle de transcription dans transcription_model.
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
    audio_plan: WorkerAudioPlanTask | Unset = UNSET
    audio_diagnostic: WorkerAudioDiagnosticInput | Unset = UNSET
    audio_dedup: WorkerAudioDedupInput | Unset = UNSET
    separation: WorkerSeparationInput | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.job import Job # noqa: PLC0415
        from ..models.transcription_pipeline import TranscriptionPipeline # noqa: PLC0415
        from ..models.worker_audio_dedup_input import WorkerAudioDedupInput # noqa: PLC0415
        from ..models.worker_audio_diagnostic_input import WorkerAudioDiagnosticInput # noqa: PLC0415
        from ..models.worker_audio_plan_task import WorkerAudioPlanTask # noqa: PLC0415
        from ..models.worker_job_input_tracks_item import WorkerJobInputTracksItem # noqa: PLC0415
        from ..models.worker_separation_input import WorkerSeparationInput # noqa: PLC0415
        from ..models.worker_vocabulary_term import WorkerVocabularyTerm # noqa: PLC0415
        from ..models.worker_voice_input import WorkerVoiceInput # noqa: PLC0415
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

        audio_plan: dict[str, Any] | Unset = UNSET
        if not isinstance(self.audio_plan, Unset):
            audio_plan = self.audio_plan.to_dict()

        audio_diagnostic: dict[str, Any] | Unset = UNSET
        if not isinstance(self.audio_diagnostic, Unset):
            audio_diagnostic = self.audio_diagnostic.to_dict()

        audio_dedup: dict[str, Any] | Unset = UNSET
        if not isinstance(self.audio_dedup, Unset):
            audio_dedup = self.audio_dedup.to_dict()

        separation: dict[str, Any] | Unset = UNSET
        if not isinstance(self.separation, Unset):
            separation = self.separation.to_dict()


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
        if audio_plan is not UNSET:
            field_dict["audio_plan"] = audio_plan
        if audio_diagnostic is not UNSET:
            field_dict["audio_diagnostic"] = audio_diagnostic
        if audio_dedup is not UNSET:
            field_dict["audio_dedup"] = audio_dedup
        if separation is not UNSET:
            field_dict["separation"] = separation

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.job import Job # noqa: PLC0415
        from ..models.transcription_pipeline import TranscriptionPipeline # noqa: PLC0415
        from ..models.worker_audio_dedup_input import WorkerAudioDedupInput # noqa: PLC0415
        from ..models.worker_audio_diagnostic_input import WorkerAudioDiagnosticInput # noqa: PLC0415
        from ..models.worker_audio_plan_task import WorkerAudioPlanTask # noqa: PLC0415
        from ..models.worker_job_input_tracks_item import WorkerJobInputTracksItem # noqa: PLC0415
        from ..models.worker_separation_input import WorkerSeparationInput # noqa: PLC0415
        from ..models.worker_vocabulary_term import WorkerVocabularyTerm # noqa: PLC0415
        from ..models.worker_voice_input import WorkerVoiceInput # noqa: PLC0415
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




        _audio_plan = d.pop("audio_plan", UNSET)
        audio_plan: WorkerAudioPlanTask | Unset
        if isinstance(_audio_plan,  Unset):
            audio_plan = UNSET
        else:
            audio_plan = WorkerAudioPlanTask.from_dict(_audio_plan)




        _audio_diagnostic = d.pop("audio_diagnostic", UNSET)
        audio_diagnostic: WorkerAudioDiagnosticInput | Unset
        if isinstance(_audio_diagnostic,  Unset):
            audio_diagnostic = UNSET
        else:
            audio_diagnostic = WorkerAudioDiagnosticInput.from_dict(_audio_diagnostic)




        _audio_dedup = d.pop("audio_dedup", UNSET)
        audio_dedup: WorkerAudioDedupInput | Unset
        if isinstance(_audio_dedup,  Unset):
            audio_dedup = UNSET
        else:
            audio_dedup = WorkerAudioDedupInput.from_dict(_audio_dedup)




        _separation = d.pop("separation", UNSET)
        separation: WorkerSeparationInput | Unset
        if isinstance(_separation,  Unset):
            separation = UNSET
        else:
            separation = WorkerSeparationInput.from_dict(_separation)




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
            audio_plan=audio_plan,
            audio_diagnostic=audio_diagnostic,
            audio_dedup=audio_dedup,
            separation=separation,
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
