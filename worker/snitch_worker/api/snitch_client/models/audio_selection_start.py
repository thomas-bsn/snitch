from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.audio_selection_request import AudioSelectionRequest





T = TypeVar("T", bound="AudioSelectionStart")



@_attrs_define
class AudioSelectionStart:
    """
        Attributes:
            selection (AudioSelectionRequest):
            preview_token (str):
            confirm_losses (bool):
     """

    selection: AudioSelectionRequest
    preview_token: str
    confirm_losses: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_selection_request import AudioSelectionRequest # noqa: PLC0415
        selection = self.selection.to_dict()

        preview_token = self.preview_token

        confirm_losses = self.confirm_losses


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "selection": selection,
            "preview_token": preview_token,
            "confirm_losses": confirm_losses,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_selection_request import AudioSelectionRequest # noqa: PLC0415
        d = dict(src_dict)
        selection = AudioSelectionRequest.from_dict(d.pop("selection"))




        preview_token = d.pop("preview_token")

        confirm_losses = d.pop("confirm_losses")

        audio_selection_start = cls(
            selection=selection,
            preview_token=preview_token,
            confirm_losses=confirm_losses,
        )

        return audio_selection_start
