from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_audio_dedup_track import WorkerAudioDedupTrack





T = TypeVar("T", bound="WorkerAudioDedupInput")



@_attrs_define
class WorkerAudioDedupInput:
    """ Tâche d'un job audio.dedup.N (comparaison acoustique des pistes retenues d'un plan). Les segments sont dans le
    repère du clip ; start_seconds est le décalage du flux de la piste.

        Attributes:
            plan_id (int):
            input_version (str):
            tracks (list[WorkerAudioDedupTrack]):
     """

    plan_id: int
    input_version: str
    tracks: list[WorkerAudioDedupTrack]





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_audio_dedup_track import WorkerAudioDedupTrack # noqa: PLC0415
        plan_id = self.plan_id

        input_version = self.input_version

        tracks = []
        for tracks_item_data in self.tracks:
            tracks_item = tracks_item_data.to_dict()
            tracks.append(tracks_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "plan_id": plan_id,
            "input_version": input_version,
            "tracks": tracks,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_audio_dedup_track import WorkerAudioDedupTrack # noqa: PLC0415
        d = dict(src_dict)
        plan_id = d.pop("plan_id")

        input_version = d.pop("input_version")

        tracks = []
        _tracks = d.pop("tracks")
        for tracks_item_data in (_tracks):
            tracks_item = WorkerAudioDedupTrack.from_dict(tracks_item_data)



            tracks.append(tracks_item)


        worker_audio_dedup_input = cls(
            plan_id=plan_id,
            input_version=input_version,
            tracks=tracks,
        )

        return worker_audio_dedup_input
