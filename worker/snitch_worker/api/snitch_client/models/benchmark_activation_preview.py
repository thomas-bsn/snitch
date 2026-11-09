from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="BenchmarkActivationPreview")



@_attrs_define
class BenchmarkActivationPreview:
    """
        Attributes:
            stage (str):
            current_model (str):
            target_model (str):
            target_label (str):
            already_active (bool):
            workers_ready (int):
            clips_with_analysis (int):
            clips_with_human_work (int): Clips portant des corrections ou validations humaines : une réanalyse les mettrait
                en jeu.
            human_items (int):
            reanalysis (str): Écran qui porte la réanalyse, qui n'est jamais lancée par l'activation.
     """

    stage: str
    current_model: str
    target_model: str
    target_label: str
    already_active: bool
    workers_ready: int
    clips_with_analysis: int
    clips_with_human_work: int
    human_items: int
    reanalysis: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        stage = self.stage

        current_model = self.current_model

        target_model = self.target_model

        target_label = self.target_label

        already_active = self.already_active

        workers_ready = self.workers_ready

        clips_with_analysis = self.clips_with_analysis

        clips_with_human_work = self.clips_with_human_work

        human_items = self.human_items

        reanalysis = self.reanalysis


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "stage": stage,
            "current_model": current_model,
            "target_model": target_model,
            "target_label": target_label,
            "already_active": already_active,
            "workers_ready": workers_ready,
            "clips_with_analysis": clips_with_analysis,
            "clips_with_human_work": clips_with_human_work,
            "human_items": human_items,
            "reanalysis": reanalysis,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        stage = d.pop("stage")

        current_model = d.pop("current_model")

        target_model = d.pop("target_model")

        target_label = d.pop("target_label")

        already_active = d.pop("already_active")

        workers_ready = d.pop("workers_ready")

        clips_with_analysis = d.pop("clips_with_analysis")

        clips_with_human_work = d.pop("clips_with_human_work")

        human_items = d.pop("human_items")

        reanalysis = d.pop("reanalysis")

        benchmark_activation_preview = cls(
            stage=stage,
            current_model=current_model,
            target_model=target_model,
            target_label=target_label,
            already_active=already_active,
            workers_ready=workers_ready,
            clips_with_analysis=clips_with_analysis,
            clips_with_human_work=clips_with_human_work,
            human_items=human_items,
            reanalysis=reanalysis,
        )


        benchmark_activation_preview.additional_properties = d
        return benchmark_activation_preview

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
