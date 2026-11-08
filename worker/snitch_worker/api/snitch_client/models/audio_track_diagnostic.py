from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audio_track_diagnostic_status import AudioTrackDiagnosticStatus
from typing import cast

if TYPE_CHECKING:
  from ..models.audio_speech_interval import AudioSpeechInterval





T = TypeVar("T", bound="AudioTrackDiagnostic")



@_attrs_define
class AudioTrackDiagnostic:
    """
        Attributes:
            plan_id (int):
            track_index (int):
            input_version (str):
            status (AudioTrackDiagnosticStatus):
            cause (str):
            detector_version (str):
            detector_config_version (str):
            intervals (list[AudioSpeechInterval]):
            job_id (int | None):
            attempt (int):
     """

    plan_id: int
    track_index: int
    input_version: str
    status: AudioTrackDiagnosticStatus
    cause: str
    detector_version: str
    detector_config_version: str
    intervals: list[AudioSpeechInterval]
    job_id: int | None
    attempt: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_speech_interval import AudioSpeechInterval # noqa: PLC0415
        plan_id = self.plan_id

        track_index = self.track_index

        input_version = self.input_version

        status = self.status.value

        cause = self.cause

        detector_version = self.detector_version

        detector_config_version = self.detector_config_version

        intervals = []
        for intervals_item_data in self.intervals:
            intervals_item = intervals_item_data.to_dict()
            intervals.append(intervals_item)



        job_id: int | None
        job_id = self.job_id

        attempt = self.attempt


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "plan_id": plan_id,
            "track_index": track_index,
            "input_version": input_version,
            "status": status,
            "cause": cause,
            "detector_version": detector_version,
            "detector_config_version": detector_config_version,
            "intervals": intervals,
            "job_id": job_id,
            "attempt": attempt,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_speech_interval import AudioSpeechInterval # noqa: PLC0415
        d = dict(src_dict)
        plan_id = d.pop("plan_id")

        track_index = d.pop("track_index")

        input_version = d.pop("input_version")

        status = AudioTrackDiagnosticStatus(d.pop("status"))




        cause = d.pop("cause")

        detector_version = d.pop("detector_version")

        detector_config_version = d.pop("detector_config_version")

        intervals = []
        _intervals = d.pop("intervals")
        for intervals_item_data in (_intervals):
            intervals_item = AudioSpeechInterval.from_dict(intervals_item_data)



            intervals.append(intervals_item)


        def _parse_job_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        job_id = _parse_job_id(d.pop("job_id"))


        attempt = d.pop("attempt")

        audio_track_diagnostic = cls(
            plan_id=plan_id,
            track_index=track_index,
            input_version=input_version,
            status=status,
            cause=cause,
            detector_version=detector_version,
            detector_config_version=detector_config_version,
            intervals=intervals,
            job_id=job_id,
            attempt=attempt,
        )

        return audio_track_diagnostic
