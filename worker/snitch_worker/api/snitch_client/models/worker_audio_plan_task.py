from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.audio_speech_interval import AudioSpeechInterval





T = TypeVar("T", bound="WorkerAudioPlanTask")



@_attrs_define
class WorkerAudioPlanTask:
    """ Tâche d’un job de transcription épinglé à un plan audio (capacité audio.multitrack.v1). Les intervalles sont
    relatifs à la piste ; le Worker ajoute start_seconds une seule fois et rend des temps sur la ligne de temps du clip.

        Attributes:
            plan_id (int):
            input_version (str):
            track_index (int):
            start_seconds (float):
            intervals (list[AudioSpeechInterval]):
     """

    plan_id: int
    input_version: str
    track_index: int
    start_seconds: float
    intervals: list[AudioSpeechInterval]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_speech_interval import AudioSpeechInterval # noqa: PLC0415
        plan_id = self.plan_id

        input_version = self.input_version

        track_index = self.track_index

        start_seconds = self.start_seconds

        intervals = []
        for intervals_item_data in self.intervals:
            intervals_item = intervals_item_data.to_dict()
            intervals.append(intervals_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "plan_id": plan_id,
            "input_version": input_version,
            "track_index": track_index,
            "start_seconds": start_seconds,
            "intervals": intervals,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_speech_interval import AudioSpeechInterval # noqa: PLC0415
        d = dict(src_dict)
        plan_id = d.pop("plan_id")

        input_version = d.pop("input_version")

        track_index = d.pop("track_index")

        start_seconds = d.pop("start_seconds")

        intervals = []
        _intervals = d.pop("intervals")
        for intervals_item_data in (_intervals):
            intervals_item = AudioSpeechInterval.from_dict(intervals_item_data)



            intervals.append(intervals_item)


        worker_audio_plan_task = cls(
            plan_id=plan_id,
            input_version=input_version,
            track_index=track_index,
            start_seconds=start_seconds,
            intervals=intervals,
        )


        worker_audio_plan_task.additional_properties = d
        return worker_audio_plan_task

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
