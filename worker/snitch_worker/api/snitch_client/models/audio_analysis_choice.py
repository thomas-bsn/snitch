from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audio_analysis_choice_mode import AudioAnalysisChoiceMode
from ..models.audio_analysis_choice_scope import AudioAnalysisChoiceScope
from typing import cast






T = TypeVar("T", bound="AudioAnalysisChoice")



@_attrs_define
class AudioAnalysisChoice:
    """
        Attributes:
            scope (AudioAnalysisChoiceScope):
            mode (AudioAnalysisChoiceMode):
            steps (list[str]):
     """

    scope: AudioAnalysisChoiceScope
    mode: AudioAnalysisChoiceMode
    steps: list[str]





    def to_dict(self) -> dict[str, Any]:
        scope = self.scope.value

        mode = self.mode.value

        steps = self.steps




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "scope": scope,
            "mode": mode,
            "steps": steps,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        scope = AudioAnalysisChoiceScope(d.pop("scope"))




        mode = AudioAnalysisChoiceMode(d.pop("mode"))




        steps = cast(list[str], d.pop("steps"))


        audio_analysis_choice = cls(
            scope=scope,
            mode=mode,
            steps=steps,
        )

        return audio_analysis_choice
