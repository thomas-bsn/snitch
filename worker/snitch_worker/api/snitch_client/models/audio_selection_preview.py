from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.audio_plan_input import AudioPlanInput
  from ..models.audio_selection_effects import AudioSelectionEffects





T = TypeVar("T", bound="AudioSelectionPreview")



@_attrs_define
class AudioSelectionPreview:
    """
        Attributes:
            input_ (AudioPlanInput):
            effects (AudioSelectionEffects):
            preview_token (str):
            expires_at (datetime.datetime):
     """

    input_: AudioPlanInput
    effects: AudioSelectionEffects
    preview_token: str
    expires_at: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_plan_input import AudioPlanInput # noqa: PLC0415
        from ..models.audio_selection_effects import AudioSelectionEffects # noqa: PLC0415
        input_ = self.input_.to_dict()

        effects = self.effects.to_dict()

        preview_token = self.preview_token

        expires_at = self.expires_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "input": input_,
            "effects": effects,
            "preview_token": preview_token,
            "expires_at": expires_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_plan_input import AudioPlanInput # noqa: PLC0415
        from ..models.audio_selection_effects import AudioSelectionEffects # noqa: PLC0415
        d = dict(src_dict)
        input_ = AudioPlanInput.from_dict(d.pop("input"))




        effects = AudioSelectionEffects.from_dict(d.pop("effects"))




        preview_token = d.pop("preview_token")

        expires_at = datetime.datetime.fromisoformat(d.pop("expires_at"))




        audio_selection_preview = cls(
            input_=input_,
            effects=effects,
            preview_token=preview_token,
            expires_at=expires_at,
        )

        return audio_selection_preview
