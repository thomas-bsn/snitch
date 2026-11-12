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
  from ..models.transcript_word import TranscriptWord





T = TypeVar("T", bound="TranscriptPart")



@_attrs_define
class TranscriptPart:
    """ Part d'un segment : des mots consécutifs du segment dits par un même Locuteur local (ADR 0065). Sans
    local_speaker_id, aucun Locuteur local ne parle sur le segment.

        Attributes:
            start (float): Début du premier mot de la part ; début du segment s'il n'a qu'une part.
            end (float): Fin du dernier mot de la part ; fin du segment s'il n'a qu'une part.
            words (list[TranscriptWord]): Mots de la part, dans l'ordre ; vide pour un segment au texte corrigé, dont le
                texte est celui du segment.
            local_speaker_id (int | Unset):
            local_speaker_label (str | Unset):
            local_speaker_decision_level (DecisionLevel | Unset): human : validation humaine ; automatic, suggestion,
                ambiguous : issus du rapprochement en base (automatic seulement lorsque l'administrateur a activé
                l'automatisation) ; unknown : aucune attribution ou aucune Personne assez proche.
            local_speaker_person_id (int | Unset): Personne du Locuteur local, si attribuée.
            local_speaker_person_name (str | Unset):
     """

    start: float
    end: float
    words: list[TranscriptWord]
    local_speaker_id: int | Unset = UNSET
    local_speaker_label: str | Unset = UNSET
    local_speaker_decision_level: DecisionLevel | Unset = UNSET
    local_speaker_person_id: int | Unset = UNSET
    local_speaker_person_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.transcript_word import TranscriptWord # noqa: PLC0415
        start = self.start

        end = self.end

        words = []
        for words_item_data in self.words:
            words_item = words_item_data.to_dict()
            words.append(words_item)



        local_speaker_id = self.local_speaker_id

        local_speaker_label = self.local_speaker_label

        local_speaker_decision_level: str | Unset = UNSET
        if not isinstance(self.local_speaker_decision_level, Unset):
            local_speaker_decision_level = self.local_speaker_decision_level.value


        local_speaker_person_id = self.local_speaker_person_id

        local_speaker_person_name = self.local_speaker_person_name


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "start": start,
            "end": end,
            "words": words,
        })
        if local_speaker_id is not UNSET:
            field_dict["local_speaker_id"] = local_speaker_id
        if local_speaker_label is not UNSET:
            field_dict["local_speaker_label"] = local_speaker_label
        if local_speaker_decision_level is not UNSET:
            field_dict["local_speaker_decision_level"] = local_speaker_decision_level
        if local_speaker_person_id is not UNSET:
            field_dict["local_speaker_person_id"] = local_speaker_person_id
        if local_speaker_person_name is not UNSET:
            field_dict["local_speaker_person_name"] = local_speaker_person_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transcript_word import TranscriptWord # noqa: PLC0415
        d = dict(src_dict)
        start = d.pop("start")

        end = d.pop("end")

        words = []
        _words = d.pop("words")
        for words_item_data in (_words):
            words_item = TranscriptWord.from_dict(words_item_data)



            words.append(words_item)


        local_speaker_id = d.pop("local_speaker_id", UNSET)

        local_speaker_label = d.pop("local_speaker_label", UNSET)

        _local_speaker_decision_level = d.pop("local_speaker_decision_level", UNSET)
        local_speaker_decision_level: DecisionLevel | Unset
        if isinstance(_local_speaker_decision_level,  Unset):
            local_speaker_decision_level = UNSET
        else:
            local_speaker_decision_level = DecisionLevel(_local_speaker_decision_level)




        local_speaker_person_id = d.pop("local_speaker_person_id", UNSET)

        local_speaker_person_name = d.pop("local_speaker_person_name", UNSET)

        transcript_part = cls(
            start=start,
            end=end,
            words=words,
            local_speaker_id=local_speaker_id,
            local_speaker_label=local_speaker_label,
            local_speaker_decision_level=local_speaker_decision_level,
            local_speaker_person_id=local_speaker_person_id,
            local_speaker_person_name=local_speaker_person_name,
        )


        transcript_part.additional_properties = d
        return transcript_part

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
