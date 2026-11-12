from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="BenchmarkRunInputChainsItem")



@_attrs_define
class BenchmarkRunInputChainsItem:
    """
        Attributes:
            diarization_model_id (str):
            transcription_model (str):
            separation_model_id (str | Unset): Facultatif : modèle de séparation du catalogue, appliqué aux Chevauchements
                où un Locuteur local reste sans paroles (#402). Rien de ce qu’il produit n’atteint la bibliothèque.
     """

    diarization_model_id: str
    transcription_model: str
    separation_model_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        diarization_model_id = self.diarization_model_id

        transcription_model = self.transcription_model

        separation_model_id = self.separation_model_id


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "diarization_model_id": diarization_model_id,
            "transcription_model": transcription_model,
        })
        if separation_model_id is not UNSET:
            field_dict["separation_model_id"] = separation_model_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        diarization_model_id = d.pop("diarization_model_id")

        transcription_model = d.pop("transcription_model")

        separation_model_id = d.pop("separation_model_id", UNSET)

        benchmark_run_input_chains_item = cls(
            diarization_model_id=diarization_model_id,
            transcription_model=transcription_model,
            separation_model_id=separation_model_id,
        )


        benchmark_run_input_chains_item.additional_properties = d
        return benchmark_run_input_chains_item

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
