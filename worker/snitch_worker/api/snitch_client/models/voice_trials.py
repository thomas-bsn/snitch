from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_trial import VoiceTrial





T = TypeVar("T", bound="VoiceTrials")



@_attrs_define
class VoiceTrials:
    """
        Attributes:
            trials (list[VoiceTrial]):
     """

    trials: list[VoiceTrial]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_trial import VoiceTrial # noqa: PLC0415
        trials = []
        for trials_item_data in self.trials:
            trials_item = trials_item_data.to_dict()
            trials.append(trials_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "trials": trials,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_trial import VoiceTrial # noqa: PLC0415
        d = dict(src_dict)
        trials = []
        _trials = d.pop("trials")
        for trials_item_data in (_trials):
            trials_item = VoiceTrial.from_dict(trials_item_data)



            trials.append(trials_item)


        voice_trials = cls(
            trials=trials,
        )


        voice_trials.additional_properties = d
        return voice_trials

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
