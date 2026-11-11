from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.voice_eligibility_input import VoiceEligibilityInput
  from ..models.voice_eligibility_requalification import VoiceEligibilityRequalification
  from ..models.voice_eligibility_rule import VoiceEligibilityRule
  from ..models.voice_eligibility_summary import VoiceEligibilitySummary





T = TypeVar("T", bound="VoiceEligibility")



@_attrs_define
class VoiceEligibility:
    """ Règle d'éligibilité des empreintes (ADR 0064) : règle active, historique, valeurs provisoires et décompte.

        Attributes:
            rule (VoiceEligibilityRule): Une version de la règle d'éligibilité. La version 1 est la règle provisoire de
                Snitch.
            history (list[VoiceEligibilityRule]): Vingt dernières versions, la plus récente d'abord.
            defaults (VoiceEligibilityInput):
            summary (VoiceEligibilitySummary): Empreintes de Locuteur local de production, par éligibilité.
            requalifications (list[VoiceEligibilityRequalification]): Dix dernières requalifications, la plus récente
                d'abord.
            requalified (int | Unset): Après une modification : empreintes dont l'éligibilité a changé.
     """

    rule: VoiceEligibilityRule
    history: list[VoiceEligibilityRule]
    defaults: VoiceEligibilityInput
    summary: VoiceEligibilitySummary
    requalifications: list[VoiceEligibilityRequalification]
    requalified: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_eligibility_input import VoiceEligibilityInput # noqa: PLC0415
        from ..models.voice_eligibility_requalification import VoiceEligibilityRequalification # noqa: PLC0415
        from ..models.voice_eligibility_rule import VoiceEligibilityRule # noqa: PLC0415
        from ..models.voice_eligibility_summary import VoiceEligibilitySummary # noqa: PLC0415
        rule = self.rule.to_dict()

        history = []
        for history_item_data in self.history:
            history_item = history_item_data.to_dict()
            history.append(history_item)



        defaults = self.defaults.to_dict()

        summary = self.summary.to_dict()

        requalifications = []
        for requalifications_item_data in self.requalifications:
            requalifications_item = requalifications_item_data.to_dict()
            requalifications.append(requalifications_item)



        requalified = self.requalified


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "rule": rule,
            "history": history,
            "defaults": defaults,
            "summary": summary,
            "requalifications": requalifications,
        })
        if requalified is not UNSET:
            field_dict["requalified"] = requalified

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_eligibility_input import VoiceEligibilityInput # noqa: PLC0415
        from ..models.voice_eligibility_requalification import VoiceEligibilityRequalification # noqa: PLC0415
        from ..models.voice_eligibility_rule import VoiceEligibilityRule # noqa: PLC0415
        from ..models.voice_eligibility_summary import VoiceEligibilitySummary # noqa: PLC0415
        d = dict(src_dict)
        rule = VoiceEligibilityRule.from_dict(d.pop("rule"))




        history = []
        _history = d.pop("history")
        for history_item_data in (_history):
            history_item = VoiceEligibilityRule.from_dict(history_item_data)



            history.append(history_item)


        defaults = VoiceEligibilityInput.from_dict(d.pop("defaults"))




        summary = VoiceEligibilitySummary.from_dict(d.pop("summary"))




        requalifications = []
        _requalifications = d.pop("requalifications")
        for requalifications_item_data in (_requalifications):
            requalifications_item = VoiceEligibilityRequalification.from_dict(requalifications_item_data)



            requalifications.append(requalifications_item)


        requalified = d.pop("requalified", UNSET)

        voice_eligibility = cls(
            rule=rule,
            history=history,
            defaults=defaults,
            summary=summary,
            requalifications=requalifications,
            requalified=requalified,
        )


        voice_eligibility.additional_properties = d
        return voice_eligibility

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
