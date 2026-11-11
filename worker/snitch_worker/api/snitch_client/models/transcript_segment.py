from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.decision_level import DecisionLevel
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.transcript_grouping import TranscriptGrouping
  from ..models.transcript_part import TranscriptPart
  from ..models.transcript_source import TranscriptSource
  from ..models.transcript_word import TranscriptWord





T = TypeVar("T", bound="TranscriptSegment")



@_attrs_define
class TranscriptSegment:
    """
        Attributes:
            start (float):
            end (float):
            text (str):
            words (list[TranscriptWord]):
            local_speaker_id (int | Unset): Locuteur local dominant du segment (celui qui y parle le plus, ou celui désigné
                par une Correction), quelle que soit la règle d'attribution ; absent sans diarisation ou sans recouvrement. Le
                détail par mots est dans parts.
            local_speaker_label (str | Unset):
            local_speaker_decision_level (DecisionLevel | Unset): human : validation humaine ; automatic, suggestion,
                ambiguous : issus du rapprochement en base (automatic seulement lorsque l'administrateur a activé
                l'automatisation) ; unknown : aucune attribution ou aucune Personne assez proche.
            local_speaker_person_id (int | Unset): Personne du Locuteur local, si attribuée.
            local_speaker_person_name (str | Unset):
            local_speaker_corrected (bool | Unset): Vrai si local_speaker_id est désigné par un utilisateur (phase 6) plutôt
                que déduit des tours de parole.
            parts (list[TranscriptPart] | Unset): Le segment lu comme une suite de parts, chacune avec son Locuteur local et
                ses mots, selon la règle d'attribution active (speaker_attribution_rule). Les parts couvrent tous les mots dans
                l'ordre ; le segment, ses bornes et son identité ne changent pas. Une seule part avec la règle par segment, pour
                un segment dont le locuteur est corrigé (elle s'applique au segment entier) ou dont le texte est corrigé. Absent
                sans diarisation.
            index (int | Unset): Rang du segment dans la transcription (identité avec transcription_job_id).
            original_text (str | Unset): Texte de la transcription automatique, jamais modifié.
            corrected (bool | Unset): Vrai si text est un texte corrigé par un utilisateur.
            correction_version (int | Unset): Version à renvoyer comme expected_version ; 0 si jamais écrit.
            corrected_by (str | Unset): Auteur de la correction du texte, si corrigé.
            corrected_at (datetime.datetime | Unset): Date de la correction du texte, si corrigé.
            sources (list[TranscriptSource] | Unset): Toutes les provenances du passage publié (piste réelle, tâche,
                segment) ; la première porte l'identité des corrections. Absent pour une ancienne transcription.
            grouping (list[TranscriptGrouping] | Unset): Sources regroupées par preuve acoustique sous ce passage,
                consultables ; absent sans regroupement.
     """

    start: float
    end: float
    text: str
    words: list[TranscriptWord]
    local_speaker_id: int | Unset = UNSET
    local_speaker_label: str | Unset = UNSET
    local_speaker_decision_level: DecisionLevel | Unset = UNSET
    local_speaker_person_id: int | Unset = UNSET
    local_speaker_person_name: str | Unset = UNSET
    local_speaker_corrected: bool | Unset = UNSET
    parts: list[TranscriptPart] | Unset = UNSET
    index: int | Unset = UNSET
    original_text: str | Unset = UNSET
    corrected: bool | Unset = UNSET
    correction_version: int | Unset = UNSET
    corrected_by: str | Unset = UNSET
    corrected_at: datetime.datetime | Unset = UNSET
    sources: list[TranscriptSource] | Unset = UNSET
    grouping: list[TranscriptGrouping] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.transcript_grouping import TranscriptGrouping # noqa: PLC0415
        from ..models.transcript_part import TranscriptPart # noqa: PLC0415
        from ..models.transcript_source import TranscriptSource # noqa: PLC0415
        from ..models.transcript_word import TranscriptWord # noqa: PLC0415
        start = self.start

        end = self.end

        text = self.text

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

        local_speaker_corrected = self.local_speaker_corrected

        parts: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.parts, Unset):
            parts = []
            for parts_item_data in self.parts:
                parts_item = parts_item_data.to_dict()
                parts.append(parts_item)



        index = self.index

        original_text = self.original_text

        corrected = self.corrected

        correction_version = self.correction_version

        corrected_by = self.corrected_by

        corrected_at: str | Unset = UNSET
        if not isinstance(self.corrected_at, Unset):
            corrected_at = self.corrected_at.isoformat()

        sources: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.sources, Unset):
            sources = []
            for sources_item_data in self.sources:
                sources_item = sources_item_data.to_dict()
                sources.append(sources_item)



        grouping: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.grouping, Unset):
            grouping = []
            for grouping_item_data in self.grouping:
                grouping_item = grouping_item_data.to_dict()
                grouping.append(grouping_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "start": start,
            "end": end,
            "text": text,
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
        if local_speaker_corrected is not UNSET:
            field_dict["local_speaker_corrected"] = local_speaker_corrected
        if parts is not UNSET:
            field_dict["parts"] = parts
        if index is not UNSET:
            field_dict["index"] = index
        if original_text is not UNSET:
            field_dict["original_text"] = original_text
        if corrected is not UNSET:
            field_dict["corrected"] = corrected
        if correction_version is not UNSET:
            field_dict["correction_version"] = correction_version
        if corrected_by is not UNSET:
            field_dict["corrected_by"] = corrected_by
        if corrected_at is not UNSET:
            field_dict["corrected_at"] = corrected_at
        if sources is not UNSET:
            field_dict["sources"] = sources
        if grouping is not UNSET:
            field_dict["grouping"] = grouping

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transcript_grouping import TranscriptGrouping # noqa: PLC0415
        from ..models.transcript_part import TranscriptPart # noqa: PLC0415
        from ..models.transcript_source import TranscriptSource # noqa: PLC0415
        from ..models.transcript_word import TranscriptWord # noqa: PLC0415
        d = dict(src_dict)
        start = d.pop("start")

        end = d.pop("end")

        text = d.pop("text")

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

        local_speaker_corrected = d.pop("local_speaker_corrected", UNSET)

        _parts = d.pop("parts", UNSET)
        parts: list[TranscriptPart] | Unset = UNSET
        if _parts is not UNSET:
            parts = []
            for parts_item_data in _parts:
                parts_item = TranscriptPart.from_dict(parts_item_data)



                parts.append(parts_item)


        index = d.pop("index", UNSET)

        original_text = d.pop("original_text", UNSET)

        corrected = d.pop("corrected", UNSET)

        correction_version = d.pop("correction_version", UNSET)

        corrected_by = d.pop("corrected_by", UNSET)

        _corrected_at = d.pop("corrected_at", UNSET)
        corrected_at: datetime.datetime | Unset
        if isinstance(_corrected_at,  Unset):
            corrected_at = UNSET
        else:
            corrected_at = datetime.datetime.fromisoformat(_corrected_at)




        _sources = d.pop("sources", UNSET)
        sources: list[TranscriptSource] | Unset = UNSET
        if _sources is not UNSET:
            sources = []
            for sources_item_data in _sources:
                sources_item = TranscriptSource.from_dict(sources_item_data)



                sources.append(sources_item)


        _grouping = d.pop("grouping", UNSET)
        grouping: list[TranscriptGrouping] | Unset = UNSET
        if _grouping is not UNSET:
            grouping = []
            for grouping_item_data in _grouping:
                grouping_item = TranscriptGrouping.from_dict(grouping_item_data)



                grouping.append(grouping_item)


        transcript_segment = cls(
            start=start,
            end=end,
            text=text,
            words=words,
            local_speaker_id=local_speaker_id,
            local_speaker_label=local_speaker_label,
            local_speaker_decision_level=local_speaker_decision_level,
            local_speaker_person_id=local_speaker_person_id,
            local_speaker_person_name=local_speaker_person_name,
            local_speaker_corrected=local_speaker_corrected,
            parts=parts,
            index=index,
            original_text=original_text,
            corrected=corrected,
            correction_version=correction_version,
            corrected_by=corrected_by,
            corrected_at=corrected_at,
            sources=sources,
            grouping=grouping,
        )


        transcript_segment.additional_properties = d
        return transcript_segment

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
