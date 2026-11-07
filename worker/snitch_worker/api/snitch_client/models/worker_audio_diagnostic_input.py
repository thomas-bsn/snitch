from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="WorkerAudioDiagnosticInput")



@_attrs_define
class WorkerAudioDiagnosticInput:
    """ Identité immuable d'un job audio.diagnostic.N. start_seconds et duration_seconds viennent de l'inventaire du plan
    (flux ffprobe), jamais du conteneur.

        Attributes:
            plan_id (int):
            track_index (int):
            input_version (str): Empreinte du plan, recopiée dans l'observation.
            detector_version (str):
            detector_config_version (str):
            start_seconds (float):
            duration_seconds (float | None):
     """

    plan_id: int
    track_index: int
    input_version: str
    detector_version: str
    detector_config_version: str
    start_seconds: float
    duration_seconds: float | None





    def to_dict(self) -> dict[str, Any]:
        plan_id = self.plan_id

        track_index = self.track_index

        input_version = self.input_version

        detector_version = self.detector_version

        detector_config_version = self.detector_config_version

        start_seconds = self.start_seconds

        duration_seconds: float | None
        duration_seconds = self.duration_seconds


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "plan_id": plan_id,
            "track_index": track_index,
            "input_version": input_version,
            "detector_version": detector_version,
            "detector_config_version": detector_config_version,
            "start_seconds": start_seconds,
            "duration_seconds": duration_seconds,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        plan_id = d.pop("plan_id")

        track_index = d.pop("track_index")

        input_version = d.pop("input_version")

        detector_version = d.pop("detector_version")

        detector_config_version = d.pop("detector_config_version")

        start_seconds = d.pop("start_seconds")

        def _parse_duration_seconds(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        duration_seconds = _parse_duration_seconds(d.pop("duration_seconds"))


        worker_audio_diagnostic_input = cls(
            plan_id=plan_id,
            track_index=track_index,
            input_version=input_version,
            detector_version=detector_version,
            detector_config_version=detector_config_version,
            start_seconds=start_seconds,
            duration_seconds=duration_seconds,
        )

        return worker_audio_diagnostic_input
