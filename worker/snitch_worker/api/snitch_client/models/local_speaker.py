from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.local_speaker_state import LocalSpeakerState
from typing import cast

if TYPE_CHECKING:
  from ..models.voice_fingerprint import VoiceFingerprint





T = TypeVar("T", bound="LocalSpeaker")



@_attrs_define
class LocalSpeaker:
    """
        Attributes:
            id (int):
            label (str):
            ordinal (int):
            speaking_seconds (float):
            turn_count (int):
            state (LocalSpeakerState): Une voix inconnue reste un Locuteur local non identifié ; les attributions arrivent
                avec les tickets suivants.
            fingerprints (list[VoiceFingerprint]):
     """

    id: int
    label: str
    ordinal: int
    speaking_seconds: float
    turn_count: int
    state: LocalSpeakerState
    fingerprints: list[VoiceFingerprint]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_fingerprint import VoiceFingerprint # noqa: PLC0415
        id = self.id

        label = self.label

        ordinal = self.ordinal

        speaking_seconds = self.speaking_seconds

        turn_count = self.turn_count

        state = self.state.value

        fingerprints = []
        for fingerprints_item_data in self.fingerprints:
            fingerprints_item = fingerprints_item_data.to_dict()
            fingerprints.append(fingerprints_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "label": label,
            "ordinal": ordinal,
            "speaking_seconds": speaking_seconds,
            "turn_count": turn_count,
            "state": state,
            "fingerprints": fingerprints,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_fingerprint import VoiceFingerprint # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        label = d.pop("label")

        ordinal = d.pop("ordinal")

        speaking_seconds = d.pop("speaking_seconds")

        turn_count = d.pop("turn_count")

        state = LocalSpeakerState(d.pop("state"))




        fingerprints = []
        _fingerprints = d.pop("fingerprints")
        for fingerprints_item_data in (_fingerprints):
            fingerprints_item = VoiceFingerprint.from_dict(fingerprints_item_data)



            fingerprints.append(fingerprints_item)


        local_speaker = cls(
            id=id,
            label=label,
            ordinal=ordinal,
            speaking_seconds=speaking_seconds,
            turn_count=turn_count,
            state=state,
            fingerprints=fingerprints,
        )


        local_speaker.additional_properties = d
        return local_speaker

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
