from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.start_clip_analysis_body_mode import StartClipAnalysisBodyMode
from ..models.start_clip_analysis_body_scope import StartClipAnalysisBodyScope
from ..models.start_clip_analysis_body_steps_item import StartClipAnalysisBodyStepsItem
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="StartClipAnalysisBody")



@_attrs_define
class StartClipAnalysisBody:
    """
        Attributes:
            scope (StartClipAnalysisBodyScope | Unset): transcription : préparation et transcription seulement. voice :
                détection des voix sur une transcription valide existante (refusée sinon, 409 transcription_required ; jamais de
                transcription implicite). full : transcription, indexation et voix. L'indexation et les voix avancent
                indépendamment après la transcription ; une branche en échec n'arrête pas l'autre et l'exécution se termine
                failed (échec partiel). Default: StartClipAnalysisBodyScope.TRANSCRIPTION.
            mode (StartClipAnalysisBodyMode | Unset): Réanalyse d'un clip déjà traité. missing : seules les étapes sans
                résultat valide sont exécutées (les résultats valides sont réutilisés). full : retranscription (périmètres
                transcription et full) et nouvelle diarisation (périmètres voice et full), l'indexation, les empreintes et la
                comparaison suivent. steps : relance les seules étapes de steps, quel que soit le résultat valide existant. Sans
                effet si une analyse est déjà active. Default: StartClipAnalysisBodyMode.MISSING.
            steps (list[StartClipAnalysisBodyStepsItem] | Unset): Avec mode steps : étapes à relancer, applicables au
                périmètre (sinon 400 invalid_reanalysis).
            confirm_replace_corrections (bool | Unset): Vrai pour confirmer qu'une retranscription rend obsolètes les
                corrections de segments du clip après succès (ADR 0012) ; leur historique est conservé. Sans lui, 409
                reanalysis_confirmation_required (stake.corrections).
            confirmed_stake_at (datetime.datetime | Unset): Le stake_at du refus reanalysis_confirmation_required que
                l'administrateur a confirmé (le plus ancien s'il y en a eu plusieurs) : le travail humain créé après n'est
                jamais rendu obsolète. Par défaut, l'instant de la requête.
            confirm_replace_validations (bool | Unset): Vrai pour confirmer qu'une nouvelle diarisation rend obsolètes les
                corrections de locuteur et détache les validations des voix après succès (#114) ; les échantillons validés et
                l'ancien résultat sont conservés. Sans lui, 409 reanalysis_confirmation_required (stake.voice).
     """

    scope: StartClipAnalysisBodyScope | Unset = StartClipAnalysisBodyScope.TRANSCRIPTION
    mode: StartClipAnalysisBodyMode | Unset = StartClipAnalysisBodyMode.MISSING
    steps: list[StartClipAnalysisBodyStepsItem] | Unset = UNSET
    confirm_replace_corrections: bool | Unset = UNSET
    confirmed_stake_at: datetime.datetime | Unset = UNSET
    confirm_replace_validations: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        scope: str | Unset = UNSET
        if not isinstance(self.scope, Unset):
            scope = self.scope.value


        mode: str | Unset = UNSET
        if not isinstance(self.mode, Unset):
            mode = self.mode.value


        steps: list[str] | Unset = UNSET
        if not isinstance(self.steps, Unset):
            steps = []
            for steps_item_data in self.steps:
                steps_item = steps_item_data.value
                steps.append(steps_item)



        confirm_replace_corrections = self.confirm_replace_corrections

        confirmed_stake_at: str | Unset = UNSET
        if not isinstance(self.confirmed_stake_at, Unset):
            confirmed_stake_at = self.confirmed_stake_at.isoformat()

        confirm_replace_validations = self.confirm_replace_validations


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
        })
        if scope is not UNSET:
            field_dict["scope"] = scope
        if mode is not UNSET:
            field_dict["mode"] = mode
        if steps is not UNSET:
            field_dict["steps"] = steps
        if confirm_replace_corrections is not UNSET:
            field_dict["confirm_replace_corrections"] = confirm_replace_corrections
        if confirmed_stake_at is not UNSET:
            field_dict["confirmed_stake_at"] = confirmed_stake_at
        if confirm_replace_validations is not UNSET:
            field_dict["confirm_replace_validations"] = confirm_replace_validations

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _scope = d.pop("scope", UNSET)
        scope: StartClipAnalysisBodyScope | Unset
        if isinstance(_scope,  Unset):
            scope = UNSET
        else:
            scope = StartClipAnalysisBodyScope(_scope)




        _mode = d.pop("mode", UNSET)
        mode: StartClipAnalysisBodyMode | Unset
        if isinstance(_mode,  Unset):
            mode = UNSET
        else:
            mode = StartClipAnalysisBodyMode(_mode)




        _steps = d.pop("steps", UNSET)
        steps: list[StartClipAnalysisBodyStepsItem] | Unset = UNSET
        if _steps is not UNSET:
            steps = []
            for steps_item_data in _steps:
                steps_item = StartClipAnalysisBodyStepsItem(steps_item_data)



                steps.append(steps_item)


        confirm_replace_corrections = d.pop("confirm_replace_corrections", UNSET)

        _confirmed_stake_at = d.pop("confirmed_stake_at", UNSET)
        confirmed_stake_at: datetime.datetime | Unset
        if isinstance(_confirmed_stake_at,  Unset):
            confirmed_stake_at = UNSET
        else:
            confirmed_stake_at = datetime.datetime.fromisoformat(_confirmed_stake_at)




        confirm_replace_validations = d.pop("confirm_replace_validations", UNSET)

        start_clip_analysis_body = cls(
            scope=scope,
            mode=mode,
            steps=steps,
            confirm_replace_corrections=confirm_replace_corrections,
            confirmed_stake_at=confirmed_stake_at,
            confirm_replace_validations=confirm_replace_validations,
        )


        start_clip_analysis_body.additional_properties = d
        return start_clip_analysis_body

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
