from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.speaker_attribution_rule_rule import SpeakerAttributionRuleRule
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.speaker_attribution_rule_version import SpeakerAttributionRuleVersion





T = TypeVar("T", bound="SpeakerAttributionRule")



@_attrs_define
class SpeakerAttributionRule:
    """ Règle d'attribution des paroles aux Locuteurs locaux (ADR 0065) : version active et historique. Elle s'applique à la
    lecture ; en changer ne réécrit aucune donnée.

        Attributes:
            version (int):
            rule (SpeakerAttributionRuleRule): word : chaque mot revient au Locuteur local dont les tours le recouvrent le
                plus, un segment se lit en parts. segment : le segment entier revient au Locuteur local qui y parle le plus
                (règle d'avant l'ADR 0065).
            author (None | str): Administrateur qui a choisi cette règle ; null pour la règle par défaut.
            created_at (datetime.datetime):
            history (list[SpeakerAttributionRuleVersion]): Vingt dernières versions, la plus récente d'abord.
     """

    version: int
    rule: SpeakerAttributionRuleRule
    author: None | str
    created_at: datetime.datetime
    history: list[SpeakerAttributionRuleVersion]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.speaker_attribution_rule_version import SpeakerAttributionRuleVersion # noqa: PLC0415
        version = self.version

        rule = self.rule.value

        author: None | str
        author = self.author

        created_at = self.created_at.isoformat()

        history = []
        for history_item_data in self.history:
            history_item = history_item_data.to_dict()
            history.append(history_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "version": version,
            "rule": rule,
            "author": author,
            "created_at": created_at,
            "history": history,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.speaker_attribution_rule_version import SpeakerAttributionRuleVersion # noqa: PLC0415
        d = dict(src_dict)
        version = d.pop("version")

        rule = SpeakerAttributionRuleRule(d.pop("rule"))




        def _parse_author(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        author = _parse_author(d.pop("author"))


        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        history = []
        _history = d.pop("history")
        for history_item_data in (_history):
            history_item = SpeakerAttributionRuleVersion.from_dict(history_item_data)



            history.append(history_item)


        speaker_attribution_rule = cls(
            version=version,
            rule=rule,
            author=author,
            created_at=created_at,
            history=history,
        )


        speaker_attribution_rule.additional_properties = d
        return speaker_attribution_rule

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
