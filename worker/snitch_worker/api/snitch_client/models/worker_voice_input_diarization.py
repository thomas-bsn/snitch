from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_voice_input_diarization_speakers_item import WorkerVoiceInputDiarizationSpeakersItem





T = TypeVar("T", bound="WorkerVoiceInputDiarization")



@_attrs_define
class WorkerVoiceInputDiarization:
    """ Étape d'empreintes uniquement : locuteurs et tours à encoder.

        Attributes:
            job_id (int):
            speakers (list[WorkerVoiceInputDiarizationSpeakersItem]):
     """

    job_id: int
    speakers: list[WorkerVoiceInputDiarizationSpeakersItem]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_voice_input_diarization_speakers_item import WorkerVoiceInputDiarizationSpeakersItem # noqa: PLC0415
        job_id = self.job_id

        speakers = []
        for speakers_item_data in self.speakers:
            speakers_item = speakers_item_data.to_dict()
            speakers.append(speakers_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "job_id": job_id,
            "speakers": speakers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_voice_input_diarization_speakers_item import WorkerVoiceInputDiarizationSpeakersItem # noqa: PLC0415
        d = dict(src_dict)
        job_id = d.pop("job_id")

        speakers = []
        _speakers = d.pop("speakers")
        for speakers_item_data in (_speakers):
            speakers_item = WorkerVoiceInputDiarizationSpeakersItem.from_dict(speakers_item_data)



            speakers.append(speakers_item)


        worker_voice_input_diarization = cls(
            job_id=job_id,
            speakers=speakers,
        )


        worker_voice_input_diarization.additional_properties = d
        return worker_voice_input_diarization

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
