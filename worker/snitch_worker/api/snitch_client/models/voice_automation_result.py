from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_matching_settings import VoiceMatchingSettings
  from ..models.voice_matching_summary import VoiceMatchingSummary





T = TypeVar("T", bound="VoiceAutomationResult")



@_attrs_define
class VoiceAutomationResult:
    """
        Attributes:
            settings (VoiceMatchingSettings):
            summary (VoiceMatchingSummary):
     """

    settings: VoiceMatchingSettings
    summary: VoiceMatchingSummary
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_matching_settings import VoiceMatchingSettings # noqa: PLC0415
        from ..models.voice_matching_summary import VoiceMatchingSummary # noqa: PLC0415
        settings = self.settings.to_dict()

        summary = self.summary.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "settings": settings,
            "summary": summary,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_matching_settings import VoiceMatchingSettings # noqa: PLC0415
        from ..models.voice_matching_summary import VoiceMatchingSummary # noqa: PLC0415
        d = dict(src_dict)
        settings = VoiceMatchingSettings.from_dict(d.pop("settings"))




        summary = VoiceMatchingSummary.from_dict(d.pop("summary"))




        voice_automation_result = cls(
            settings=settings,
            summary=summary,
        )


        voice_automation_result.additional_properties = d
        return voice_automation_result

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
