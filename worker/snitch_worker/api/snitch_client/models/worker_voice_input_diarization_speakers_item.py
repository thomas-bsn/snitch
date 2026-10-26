from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_voice_input_diarization_speakers_item_turns_item import WorkerVoiceInputDiarizationSpeakersItemTurnsItem





T = TypeVar("T", bound="WorkerVoiceInputDiarizationSpeakersItem")



@_attrs_define
class WorkerVoiceInputDiarizationSpeakersItem:
    """
        Attributes:
            label (str):
            turns (list[WorkerVoiceInputDiarizationSpeakersItemTurnsItem]):
     """

    label: str
    turns: list[WorkerVoiceInputDiarizationSpeakersItemTurnsItem]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_voice_input_diarization_speakers_item_turns_item import WorkerVoiceInputDiarizationSpeakersItemTurnsItem # noqa: PLC0415
        label = self.label

        turns = []
        for turns_item_data in self.turns:
            turns_item = turns_item_data.to_dict()
            turns.append(turns_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "label": label,
            "turns": turns,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_voice_input_diarization_speakers_item_turns_item import WorkerVoiceInputDiarizationSpeakersItemTurnsItem # noqa: PLC0415
        d = dict(src_dict)
        label = d.pop("label")

        turns = []
        _turns = d.pop("turns")
        for turns_item_data in (_turns):
            turns_item = WorkerVoiceInputDiarizationSpeakersItemTurnsItem.from_dict(turns_item_data)



            turns.append(turns_item)


        worker_voice_input_diarization_speakers_item = cls(
            label=label,
            turns=turns,
        )


        worker_voice_input_diarization_speakers_item.additional_properties = d
        return worker_voice_input_diarization_speakers_item

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
