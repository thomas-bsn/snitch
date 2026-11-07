from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audio_selection_request_mode import AudioSelectionRequestMode
from typing import cast

if TYPE_CHECKING:
  from ..models.audio_analysis_choice import AudioAnalysisChoice
  from ..models.audio_selection_decision import AudioSelectionDecision





T = TypeVar("T", bound="AudioSelectionRequest")



@_attrs_define
class AudioSelectionRequest:
    """
        Attributes:
            version (str):
            mode (AudioSelectionRequestMode):
            track_indices (list[int]):
            decisions (list[AudioSelectionDecision]):
            analysis (AudioAnalysisChoice):
     """

    version: str
    mode: AudioSelectionRequestMode
    track_indices: list[int]
    decisions: list[AudioSelectionDecision]
    analysis: AudioAnalysisChoice





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_analysis_choice import AudioAnalysisChoice # noqa: PLC0415
        from ..models.audio_selection_decision import AudioSelectionDecision # noqa: PLC0415
        version = self.version

        mode = self.mode.value

        track_indices = self.track_indices



        decisions = []
        for decisions_item_data in self.decisions:
            decisions_item = decisions_item_data.to_dict()
            decisions.append(decisions_item)



        analysis = self.analysis.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "version": version,
            "mode": mode,
            "track_indices": track_indices,
            "decisions": decisions,
            "analysis": analysis,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_analysis_choice import AudioAnalysisChoice # noqa: PLC0415
        from ..models.audio_selection_decision import AudioSelectionDecision # noqa: PLC0415
        d = dict(src_dict)
        version = d.pop("version")

        mode = AudioSelectionRequestMode(d.pop("mode"))




        track_indices = cast(list[int], d.pop("track_indices"))


        decisions = []
        _decisions = d.pop("decisions")
        for decisions_item_data in (_decisions):
            decisions_item = AudioSelectionDecision.from_dict(decisions_item_data)



            decisions.append(decisions_item)


        analysis = AudioAnalysisChoice.from_dict(d.pop("analysis"))




        audio_selection_request = cls(
            version=version,
            mode=mode,
            track_indices=track_indices,
            decisions=decisions,
            analysis=analysis,
        )

        return audio_selection_request
