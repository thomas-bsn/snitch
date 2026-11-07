from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audio_plan_input_mode import AudioPlanInputMode
from typing import cast

if TYPE_CHECKING:
  from ..models.audio_track_decision import AudioTrackDecision
  from ..models.audio_track_inventory import AudioTrackInventory





T = TypeVar("T", bound="AudioPlanInput")



@_attrs_define
class AudioPlanInput:
    """
        Attributes:
            mode (AudioPlanInputMode):
            track_indices (list[int]):
            inventory (list[AudioTrackInventory]):
            decisions (list[AudioTrackDecision]):
            detector_version (str):
            detector_config_version (str):
            merge_version (str):
            pipeline_version (str):
            vocabulary_version (str):
     """

    mode: AudioPlanInputMode
    track_indices: list[int]
    inventory: list[AudioTrackInventory]
    decisions: list[AudioTrackDecision]
    detector_version: str
    detector_config_version: str
    merge_version: str
    pipeline_version: str
    vocabulary_version: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_track_decision import AudioTrackDecision # noqa: PLC0415
        from ..models.audio_track_inventory import AudioTrackInventory # noqa: PLC0415
        mode = self.mode.value

        track_indices = self.track_indices



        inventory = []
        for inventory_item_data in self.inventory:
            inventory_item = inventory_item_data.to_dict()
            inventory.append(inventory_item)



        decisions = []
        for decisions_item_data in self.decisions:
            decisions_item = decisions_item_data.to_dict()
            decisions.append(decisions_item)



        detector_version = self.detector_version

        detector_config_version = self.detector_config_version

        merge_version = self.merge_version

        pipeline_version = self.pipeline_version

        vocabulary_version = self.vocabulary_version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
            "track_indices": track_indices,
            "inventory": inventory,
            "decisions": decisions,
            "detector_version": detector_version,
            "detector_config_version": detector_config_version,
            "merge_version": merge_version,
            "pipeline_version": pipeline_version,
            "vocabulary_version": vocabulary_version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_track_decision import AudioTrackDecision # noqa: PLC0415
        from ..models.audio_track_inventory import AudioTrackInventory # noqa: PLC0415
        d = dict(src_dict)
        mode = AudioPlanInputMode(d.pop("mode"))




        track_indices = cast(list[int], d.pop("track_indices"))


        inventory = []
        _inventory = d.pop("inventory")
        for inventory_item_data in (_inventory):
            inventory_item = AudioTrackInventory.from_dict(inventory_item_data)



            inventory.append(inventory_item)


        decisions = []
        _decisions = d.pop("decisions")
        for decisions_item_data in (_decisions):
            decisions_item = AudioTrackDecision.from_dict(decisions_item_data)



            decisions.append(decisions_item)


        detector_version = d.pop("detector_version")

        detector_config_version = d.pop("detector_config_version")

        merge_version = d.pop("merge_version")

        pipeline_version = d.pop("pipeline_version")

        vocabulary_version = d.pop("vocabulary_version")

        audio_plan_input = cls(
            mode=mode,
            track_indices=track_indices,
            inventory=inventory,
            decisions=decisions,
            detector_version=detector_version,
            detector_config_version=detector_config_version,
            merge_version=merge_version,
            pipeline_version=pipeline_version,
            vocabulary_version=vocabulary_version,
        )

        return audio_plan_input
