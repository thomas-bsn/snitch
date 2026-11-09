from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="BenchmarkActivationResult")



@_attrs_define
class BenchmarkActivationResult:
    """
        Attributes:
            version (str):
            stage (str):
            model (str):
            reanalysis (str):
            clips_with_human_work (int):
     """

    version: str
    stage: str
    model: str
    reanalysis: str
    clips_with_human_work: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        version = self.version

        stage = self.stage

        model = self.model

        reanalysis = self.reanalysis

        clips_with_human_work = self.clips_with_human_work


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "version": version,
            "stage": stage,
            "model": model,
            "reanalysis": reanalysis,
            "clips_with_human_work": clips_with_human_work,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version")

        stage = d.pop("stage")

        model = d.pop("model")

        reanalysis = d.pop("reanalysis")

        clips_with_human_work = d.pop("clips_with_human_work")

        benchmark_activation_result = cls(
            version=version,
            stage=stage,
            model=model,
            reanalysis=reanalysis,
            clips_with_human_work=clips_with_human_work,
        )


        benchmark_activation_result.additional_properties = d
        return benchmark_activation_result

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
