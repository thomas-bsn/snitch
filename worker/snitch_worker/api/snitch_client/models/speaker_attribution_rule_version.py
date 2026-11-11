from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.speaker_attribution_rule_version_rule import SpeakerAttributionRuleVersionRule
from typing import cast
import datetime






T = TypeVar("T", bound="SpeakerAttributionRuleVersion")



@_attrs_define
class SpeakerAttributionRuleVersion:
    """
        Attributes:
            version (int):
            rule (SpeakerAttributionRuleVersionRule): word : chaque mot revient au Locuteur local dont les tours le
                recouvrent le plus, un segment se lit en parts. segment : le segment entier revient au Locuteur local qui y
                parle le plus (règle d'avant l'ADR 0065).
            author (None | str): Administrateur qui a choisi cette règle ; null pour la règle par défaut.
            created_at (datetime.datetime):
     """

    version: int
    rule: SpeakerAttributionRuleVersionRule
    author: None | str
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        version = self.version

        rule = self.rule.value

        author: None | str
        author = self.author

        created_at = self.created_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "version": version,
            "rule": rule,
            "author": author,
            "created_at": created_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version")

        rule = SpeakerAttributionRuleVersionRule(d.pop("rule"))




        def _parse_author(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        author = _parse_author(d.pop("author"))


        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        speaker_attribution_rule_version = cls(
            version=version,
            rule=rule,
            author=author,
            created_at=created_at,
        )


        speaker_attribution_rule_version.additional_properties = d
        return speaker_attribution_rule_version

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
