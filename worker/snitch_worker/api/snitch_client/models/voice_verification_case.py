from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_verification_case_decision_level import VoiceVerificationCaseDecisionLevel
from typing import cast

if TYPE_CHECKING:
  from ..models.attribution_candidate import AttributionCandidate





T = TypeVar("T", bound="VoiceVerificationCase")



@_attrs_define
class VoiceVerificationCase:
    """
        Attributes:
            local_speaker_id (int): Extrait écoutable par GET /api/local-speakers/{id}/excerpt ; se tranche par POST
                /api/local-speakers/{id}/validation.
            clip_id (int):
            clip_name (str):
            label (str):
            speaking_seconds (float):
            decision_level (VoiceVerificationCaseDecisionLevel):
            candidates (list[AttributionCandidate]): Personnes entre lesquelles une voix ambiguë hésite ; vide pour une voix
                inconnue.
            group_id (int): Groupe proposé de la voix (voir /api/voice-groups), la voix seule si aucun lien.
            group_voice_count (int): Voix inconnues visibles du groupe, calculé en base.
            group_clip_count (int): Clips distincts du groupe : la fréquence qui ordonne les cas.
     """

    local_speaker_id: int
    clip_id: int
    clip_name: str
    label: str
    speaking_seconds: float
    decision_level: VoiceVerificationCaseDecisionLevel
    candidates: list[AttributionCandidate]
    group_id: int
    group_voice_count: int
    group_clip_count: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.attribution_candidate import AttributionCandidate # noqa: PLC0415
        local_speaker_id = self.local_speaker_id

        clip_id = self.clip_id

        clip_name = self.clip_name

        label = self.label

        speaking_seconds = self.speaking_seconds

        decision_level = self.decision_level.value

        candidates = []
        for candidates_item_data in self.candidates:
            candidates_item = candidates_item_data.to_dict()
            candidates.append(candidates_item)



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
            "candidates": candidates,
            "group_id": group_id,
            "group_voice_count": group_voice_count,
            "group_clip_count": group_clip_count,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.attribution_candidate import AttributionCandidate # noqa: PLC0415
        d = dict(src_dict)
        local_speaker_id = d.pop("local_speaker_id")

        clip_id = d.pop("clip_id")

        clip_name = d.pop("clip_name")

        label = d.pop("label")

        speaking_seconds = d.pop("speaking_seconds")

        decision_level = VoiceVerificationCaseDecisionLevel(d.pop("decision_level"))




        candidates = []
        _candidates = d.pop("candidates")
        for candidates_item_data in (_candidates):
            candidates_item = AttributionCandidate.from_dict(candidates_item_data)



            candidates.append(candidates_item)


        group_id = d.pop("group_id")

        group_voice_count = d.pop("group_voice_count")

        group_clip_count = d.pop("group_clip_count")

        voice_verification_case = cls(
            local_speaker_id=local_speaker_id,
            clip_id=clip_id,
            clip_name=clip_name,
            label=label,
            speaking_seconds=speaking_seconds,
            decision_level=decision_level,
            candidates=candidates,
            group_id=group_id,
            group_voice_count=group_voice_count,
            group_clip_count=group_clip_count,
        )


        voice_verification_case.additional_properties = d
        return voice_verification_case

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
