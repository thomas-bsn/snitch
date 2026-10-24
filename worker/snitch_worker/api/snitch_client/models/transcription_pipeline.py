from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transcription_pipeline_preparation import TranscriptionPipelinePreparation
from ..models.transcription_pipeline_provider import TranscriptionPipelineProvider
from typing import cast

if TYPE_CHECKING:
  from ..models.transcription_pipeline_vad import TranscriptionPipelineVad





T = TypeVar("T", bound="TranscriptionPipeline")



@_attrs_define
class TranscriptionPipeline:
    """
        Attributes:
            provider (TranscriptionPipelineProvider):
            model (str):
            language (str):
            preparation (TranscriptionPipelinePreparation):
            vad (TranscriptionPipelineVad):
     """

    provider: TranscriptionPipelineProvider
    model: str
    language: str
    preparation: TranscriptionPipelinePreparation
    vad: TranscriptionPipelineVad





    def to_dict(self) -> dict[str, Any]:
        from ..models.transcription_pipeline_vad import TranscriptionPipelineVad # noqa: PLC0415
        provider = self.provider.value

        model = self.model

        language = self.language

        preparation = self.preparation.value

        vad = self.vad.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "provider": provider,
            "model": model,
            "language": language,
            "preparation": preparation,
            "vad": vad,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transcription_pipeline_vad import TranscriptionPipelineVad # noqa: PLC0415
        d = dict(src_dict)
        provider = TranscriptionPipelineProvider(d.pop("provider"))




        model = d.pop("model")

        language = d.pop("language")

        preparation = TranscriptionPipelinePreparation(d.pop("preparation"))




        vad = TranscriptionPipelineVad.from_dict(d.pop("vad"))




        transcription_pipeline = cls(
            provider=provider,
            model=model,
            language=language,
            preparation=preparation,
            vad=vad,
        )

        return transcription_pipeline
