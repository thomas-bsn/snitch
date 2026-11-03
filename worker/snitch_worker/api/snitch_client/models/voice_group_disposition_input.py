from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_group_disposition_input_disposition import VoiceGroupDispositionInputDisposition
from typing import cast






T = TypeVar("T", bound="VoiceGroupDispositionInput")



@_attrs_define
class VoiceGroupDispositionInput:
    """
        Attributes:
            disposition (VoiceGroupDispositionInputDisposition):
            local_speaker_ids (list[int]): Les voix du groupe affichées (non exclues) ; group_changed si le groupe n'est
                plus exactement cet ensemble.
     """

    disposition: VoiceGroupDispositionInputDisposition
    local_speaker_ids: list[int]





    def to_dict(self) -> dict[str, Any]:
        disposition = self.disposition.value

        local_speaker_ids = self.local_speaker_ids




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "disposition": disposition,
            "local_speaker_ids": local_speaker_ids,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        disposition = VoiceGroupDispositionInputDisposition(d.pop("disposition"))




        local_speaker_ids = cast(list[int], d.pop("local_speaker_ids"))


        voice_group_disposition_input = cls(
            disposition=disposition,
            local_speaker_ids=local_speaker_ids,
        )

        return voice_group_disposition_input
