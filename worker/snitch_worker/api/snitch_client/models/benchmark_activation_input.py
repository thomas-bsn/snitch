from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.benchmark_activation_input_stage import BenchmarkActivationInputStage
from ..types import UNSET, Unset






T = TypeVar("T", bound="BenchmarkActivationInput")



@_attrs_define
class BenchmarkActivationInput:
    """
        Attributes:
            stage (BenchmarkActivationInputStage):
            model (str):
            expected_current (str): Le modèle remplacé tel que l'aperçu l'a montré : un changement survenu depuis est refusé
                (benchmark_activation_stale).
            confirm (bool):
            run_id (int | Unset):
     """

    stage: BenchmarkActivationInputStage
    model: str
    expected_current: str
    confirm: bool
    run_id: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        stage = self.stage.value

        model = self.model

        expected_current = self.expected_current

        confirm = self.confirm

        run_id = self.run_id


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "stage": stage,
            "model": model,
            "expected_current": expected_current,
            "confirm": confirm,
        })
        if run_id is not UNSET:
            field_dict["run_id"] = run_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        stage = BenchmarkActivationInputStage(d.pop("stage"))




        model = d.pop("model")

        expected_current = d.pop("expected_current")

        confirm = d.pop("confirm")

        run_id = d.pop("run_id", UNSET)

        benchmark_activation_input = cls(
            stage=stage,
            model=model,
            expected_current=expected_current,
            confirm=confirm,
            run_id=run_id,
        )


        benchmark_activation_input.additional_properties = d
        return benchmark_activation_input

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
