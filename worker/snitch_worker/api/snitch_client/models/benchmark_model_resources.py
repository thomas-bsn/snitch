from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="BenchmarkModelResources")



@_attrs_define
class BenchmarkModelResources:
    """
        Attributes:
            model_id (str):
            tasks (int): Tâches terminées du modèle dans le run.
            inference_seconds (float | None): Durée cumulée des tâches sur le Worker, téléchargement et préparation de
                l'audio exclus : l'inférence du modèle et, pour une séparation, la transcription et l'empreinte des sources ;
                null si une tâche ne l'a pas remontée (Worker ancien), jamais zéro par défaut.
            gpu_peak_bytes (int | None): Pic de mémoire GPU allouée par PyTorch sur ces tâches ; null sans CUDA, pour la
                transcription (service externe) ou si une tâche ne l'a pas remonté.
     """

    model_id: str
    tasks: int
    inference_seconds: float | None
    gpu_peak_bytes: int | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        model_id = self.model_id

        tasks = self.tasks

        inference_seconds: float | None
        inference_seconds = self.inference_seconds

        gpu_peak_bytes: int | None
        gpu_peak_bytes = self.gpu_peak_bytes


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "model_id": model_id,
            "tasks": tasks,
            "inference_seconds": inference_seconds,
            "gpu_peak_bytes": gpu_peak_bytes,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        model_id = d.pop("model_id")

        tasks = d.pop("tasks")

        def _parse_inference_seconds(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        inference_seconds = _parse_inference_seconds(d.pop("inference_seconds"))


        def _parse_gpu_peak_bytes(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        gpu_peak_bytes = _parse_gpu_peak_bytes(d.pop("gpu_peak_bytes"))


        benchmark_model_resources = cls(
            model_id=model_id,
            tasks=tasks,
            inference_seconds=inference_seconds,
            gpu_peak_bytes=gpu_peak_bytes,
        )


        benchmark_model_resources.additional_properties = d
        return benchmark_model_resources

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
