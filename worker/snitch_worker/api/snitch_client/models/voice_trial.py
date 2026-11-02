from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_trial_status import VoiceTrialStatus
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.voice_trial_speaker import VoiceTrialSpeaker





T = TypeVar("T", bound="VoiceTrial")



@_attrs_define
class VoiceTrial:
    """
        Attributes:
            id (int):
            clip_id (int):
            audio_stream_index (int):
            model_id (str):
            num_speakers (int | None):
            min_speakers (int | None):
            max_speakers (int | None):
            status (VoiceTrialStatus):
            speakers (list[VoiceTrialSpeaker]):
            created_at (str | Unset):
     """

    id: int
    clip_id: int
    audio_stream_index: int
    model_id: str
    num_speakers: int | None
    min_speakers: int | None
    max_speakers: int | None
    status: VoiceTrialStatus
    speakers: list[VoiceTrialSpeaker]
    created_at: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_trial_speaker import VoiceTrialSpeaker # noqa: PLC0415
        id = self.id

        clip_id = self.clip_id

        audio_stream_index = self.audio_stream_index

        model_id = self.model_id

        num_speakers: int | None
        num_speakers = self.num_speakers

        min_speakers: int | None
        min_speakers = self.min_speakers

        max_speakers: int | None
        max_speakers = self.max_speakers

        status = self.status.value

        speakers = []
        for speakers_item_data in self.speakers:
            speakers_item = speakers_item_data.to_dict()
            speakers.append(speakers_item)



        created_at = self.created_at


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "clip_id": clip_id,
            "audio_stream_index": audio_stream_index,
            "model_id": model_id,
            "num_speakers": num_speakers,
            "min_speakers": min_speakers,
            "max_speakers": max_speakers,
            "status": status,
            "speakers": speakers,
        })
        if created_at is not UNSET:
            field_dict["created_at"] = created_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_trial_speaker import VoiceTrialSpeaker # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        clip_id = d.pop("clip_id")

        audio_stream_index = d.pop("audio_stream_index")

        model_id = d.pop("model_id")

        def _parse_num_speakers(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        num_speakers = _parse_num_speakers(d.pop("num_speakers"))


        def _parse_min_speakers(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        min_speakers = _parse_min_speakers(d.pop("min_speakers"))


        def _parse_max_speakers(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        max_speakers = _parse_max_speakers(d.pop("max_speakers"))


        status = VoiceTrialStatus(d.pop("status"))




        speakers = []
        _speakers = d.pop("speakers")
        for speakers_item_data in (_speakers):
            speakers_item = VoiceTrialSpeaker.from_dict(speakers_item_data)



            speakers.append(speakers_item)


        created_at = d.pop("created_at", UNSET)

        voice_trial = cls(
            id=id,
            clip_id=clip_id,
            audio_stream_index=audio_stream_index,
            model_id=model_id,
            num_speakers=num_speakers,
            min_speakers=min_speakers,
            max_speakers=max_speakers,
            status=status,
            speakers=speakers,
            created_at=created_at,
        )


        voice_trial.additional_properties = d
        return voice_trial

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
