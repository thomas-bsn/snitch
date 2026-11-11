from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.voice_eligibility_outcome import VoiceEligibilityOutcome





T = TypeVar("T", bound="VoiceEligibilityRequalification")



@_attrs_define
class VoiceEligibilityRequalification:
    """ Une requalification tracée : auteur, règle appliquée, date et rapport avant/après.

        Attributes:
            id (int):
            rule_version (int):
            author (None | str):
            created_at (datetime.datetime):
            changed (int): Empreintes de production dont l'éligibilité a changé.
            missing_sources (int): Empreintes requalifiées dont le clip n'a plus de source audio connue.
            estimated (int): Empreintes à Parole nette estimée.
            original_vectors (int): Empreintes gardant leur vecteur d'origine.
            before (VoiceEligibilityOutcome): Empreintes de Locuteur local de production par éligibilité, et Personnes qui
                ont une Référence.
            after (VoiceEligibilityOutcome): Empreintes de Locuteur local de production par éligibilité, et Personnes qui
                ont une Référence.
     """

    id: int
    rule_version: int
    author: None | str
    created_at: datetime.datetime
    changed: int
    missing_sources: int
    estimated: int
    original_vectors: int
    before: VoiceEligibilityOutcome
    after: VoiceEligibilityOutcome
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_eligibility_outcome import VoiceEligibilityOutcome # noqa: PLC0415
        id = self.id

        rule_version = self.rule_version

        author: None | str
        author = self.author

        created_at = self.created_at.isoformat()

        changed = self.changed

        missing_sources = self.missing_sources

        estimated = self.estimated

        original_vectors = self.original_vectors

        before = self.before.to_dict()

        after = self.after.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "rule_version": rule_version,
            "author": author,
            "created_at": created_at,
            "changed": changed,
            "missing_sources": missing_sources,
            "estimated": estimated,
            "original_vectors": original_vectors,
            "before": before,
            "after": after,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_eligibility_outcome import VoiceEligibilityOutcome # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        rule_version = d.pop("rule_version")

        def _parse_author(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        author = _parse_author(d.pop("author"))


        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        changed = d.pop("changed")

        missing_sources = d.pop("missing_sources")

        estimated = d.pop("estimated")

        original_vectors = d.pop("original_vectors")

        before = VoiceEligibilityOutcome.from_dict(d.pop("before"))




        after = VoiceEligibilityOutcome.from_dict(d.pop("after"))




        voice_eligibility_requalification = cls(
            id=id,
            rule_version=rule_version,
            author=author,
            created_at=created_at,
            changed=changed,
            missing_sources=missing_sources,
            estimated=estimated,
            original_vectors=original_vectors,
            before=before,
            after=after,
        )


        voice_eligibility_requalification.additional_properties = d
        return voice_eligibility_requalification

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
