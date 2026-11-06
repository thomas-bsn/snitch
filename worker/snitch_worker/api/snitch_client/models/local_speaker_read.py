from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.decision_level import DecisionLevel
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.attribution import Attribution





T = TypeVar("T", bound="LocalSpeakerRead")



@_attrs_define
class LocalSpeakerRead:
    """
        Attributes:
            local_speaker_id (int):
            clip_id (int):
            clip_name (str):
            label (str):
            speaking_seconds (float):
            decision_level (DecisionLevel): human : validation humaine ; automatic, suggestion, ambiguous : issus du
                rapprochement en base (automatic seulement lorsque l'administrateur a activé l'automatisation) ; unknown :
                aucune attribution ou aucune Personne assez proche.
            attribution (Attribution | None): Attribution effective (validation humaine d'abord, sinon rapprochement) ; null
                sans attribution. Author et decided_at disent qui l'a décidée et quand.
            group_id (int | Unset): Présent seulement pour une voix à trancher (inconnue ou ambiguë) : groupe proposé de la
                voix.
            group_voice_count (int | Unset): Présent avec group_id.
            group_clip_count (int | Unset): Présent avec group_id.
     """

    local_speaker_id: int
    clip_id: int
    clip_name: str
    label: str
    speaking_seconds: float
    decision_level: DecisionLevel
    attribution: Attribution | None
    group_id: int | Unset = UNSET
    group_voice_count: int | Unset = UNSET
    group_clip_count: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.attribution import Attribution # noqa: PLC0415
        local_speaker_id = self.local_speaker_id

        clip_id = self.clip_id

        clip_name = self.clip_name

        label = self.label

        speaking_seconds = self.speaking_seconds

        decision_level = self.decision_level.value

        attribution: dict[str, Any] | None
        if isinstance(self.attribution, Attribution):
            attribution = self.attribution.to_dict()
        else:
            attribution = self.attribution

        group_id = self.group_id

        group_voice_count = self.group_voice_count

        group_clip_count = self.group_clip_count


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "local_speaker_id": local_speaker_id,
            "clip_id": clip_id,
            "clip_name": clip_name,
            "label": label,
            "speaking_seconds": speaking_seconds,
            "decision_level": decision_level,
            "attribution": attribution,
        })
        if group_id is not UNSET:
            field_dict["group_id"] = group_id
        if group_voice_count is not UNSET:
            field_dict["group_voice_count"] = group_voice_count
        if group_clip_count is not UNSET:
            field_dict["group_clip_count"] = group_clip_count

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.attribution import Attribution # noqa: PLC0415
        d = dict(src_dict)
        local_speaker_id = d.pop("local_speaker_id")

        clip_id = d.pop("clip_id")

        clip_name = d.pop("clip_name")

        label = d.pop("label")

        speaking_seconds = d.pop("speaking_seconds")

        decision_level = DecisionLevel(d.pop("decision_level"))




        def _parse_attribution(data: object) -> Attribution | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                attribution_type_1 = Attribution.from_dict(data)



                return attribution_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Attribution | None, data)

        attribution = _parse_attribution(d.pop("attribution"))


        group_id = d.pop("group_id", UNSET)

        group_voice_count = d.pop("group_voice_count", UNSET)

        group_clip_count = d.pop("group_clip_count", UNSET)

        local_speaker_read = cls(
            local_speaker_id=local_speaker_id,
            clip_id=clip_id,
            clip_name=clip_name,
            label=label,
            speaking_seconds=speaking_seconds,
            decision_level=decision_level,
            attribution=attribution,
            group_id=group_id,
            group_voice_count=group_voice_count,
            group_clip_count=group_clip_count,
        )


        local_speaker_read.additional_properties = d
        return local_speaker_read

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
