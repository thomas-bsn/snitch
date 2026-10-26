from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.voice_pipeline import VoicePipeline
  from ..models.worker_voice_input_diarization import WorkerVoiceInputDiarization





T = TypeVar("T", bound="WorkerVoiceInput")



@_attrs_define
class WorkerVoiceInput:
    """
        Attributes:
            pipeline_version (str):
            config (VoicePipeline):
            diarization (WorkerVoiceInputDiarization | Unset): Étape d'empreintes uniquement : locuteurs et tours à encoder.
     """

    pipeline_version: str
    config: VoicePipeline
    diarization: WorkerVoiceInputDiarization | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_pipeline import VoicePipeline # noqa: PLC0415
        from ..models.worker_voice_input_diarization import WorkerVoiceInputDiarization # noqa: PLC0415
        pipeline_version = self.pipeline_version

        config = self.config.to_dict()

        diarization: dict[str, Any] | Unset = UNSET
        if not isinstance(self.diarization, Unset):
            diarization = self.diarization.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "pipeline_version": pipeline_version,
            "config": config,
        })
        if diarization is not UNSET:
            field_dict["diarization"] = diarization

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_pipeline import VoicePipeline # noqa: PLC0415
        from ..models.worker_voice_input_diarization import WorkerVoiceInputDiarization # noqa: PLC0415
        d = dict(src_dict)
        pipeline_version = d.pop("pipeline_version")

        config = VoicePipeline.from_dict(d.pop("config"))




        _diarization = d.pop("diarization", UNSET)
        diarization: WorkerVoiceInputDiarization | Unset
        if isinstance(_diarization,  Unset):
            diarization = UNSET
        else:
            diarization = WorkerVoiceInputDiarization.from_dict(_diarization)




        worker_voice_input = cls(
            pipeline_version=pipeline_version,
            config=config,
            diarization=diarization,
        )


        worker_voice_input.additional_properties = d
        return worker_voice_input

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
