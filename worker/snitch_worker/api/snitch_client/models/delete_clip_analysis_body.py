from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.delete_clip_analysis_body_steps_item import DeleteClipAnalysisBodyStepsItem
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="DeleteClipAnalysisBody")



@_attrs_define
class DeleteClipAnalysisBody:
    """
        Attributes:
            steps (list[DeleteClipAnalysisBodyStepsItem] | Unset): Étapes à supprimer, avec ce qui dépend d'elles (ADR 0030)
                : transcription supprime l'indexation et les corrections ; diarisation supprime les voix, leurs empreintes, le
                rapprochement et les identifications, y compris les voix validées (leurs échantillons quittent la référence ;
                les Personnes restent) ; empreintes supprime le rapprochement.
            all_ (bool | Unset): Vrai pour supprimer toute l'analyse : les étapes ci-dessus, l'historique des analyses, les
                essais de diarisation et les notifications du clip. Remplace steps.
            confirm_loss (bool | Unset): Vrai pour confirmer les pertes indiquées par le calcul préalable (stake). Sans lui,
                409 analysis_deletion_confirmation_required quand la suppression perd quelque chose.
     """

    steps: list[DeleteClipAnalysisBodyStepsItem] | Unset = UNSET
    all_: bool | Unset = UNSET
    confirm_loss: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        steps: list[str] | Unset = UNSET
        if not isinstance(self.steps, Unset):
            steps = []
            for steps_item_data in self.steps:
                steps_item = steps_item_data.value
                steps.append(steps_item)



        all_ = self.all_

        confirm_loss = self.confirm_loss


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
        })
        if steps is not UNSET:
            field_dict["steps"] = steps
        if all_ is not UNSET:
            field_dict["all"] = all_
        if confirm_loss is not UNSET:
            field_dict["confirm_loss"] = confirm_loss

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _steps = d.pop("steps", UNSET)
        steps: list[DeleteClipAnalysisBodyStepsItem] | Unset = UNSET
        if _steps is not UNSET:
            steps = []
            for steps_item_data in _steps:
                steps_item = DeleteClipAnalysisBodyStepsItem(steps_item_data)



                steps.append(steps_item)


        all_ = d.pop("all", UNSET)

        confirm_loss = d.pop("confirm_loss", UNSET)

        delete_clip_analysis_body = cls(
            steps=steps,
            all_=all_,
            confirm_loss=confirm_loss,
        )


        delete_clip_analysis_body.additional_properties = d
        return delete_clip_analysis_body

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
