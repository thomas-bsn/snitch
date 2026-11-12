from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_separation_input_segments_item import WorkerSeparationInputSegmentsItem





T = TypeVar("T", bound="WorkerSeparationInput")



@_attrs_define
class WorkerSeparationInput:
    """ Tâche benchmark.separation.N (#402) : le modèle de séparation demandé et la transcription de la chaîne, dans le
    repère de la piste. Les tours sont dans voice.diarization, le modèle de transcription dans transcription_model.

        Attributes:
            model (str):
            revision (str):
            ambiguity_margin (float): Écart minimal de ressemblance entre les deux meilleurs Locuteurs locaux pour attribuer
                une Source séparée.
            segments (list[WorkerSeparationInputSegmentsItem]):
     """

    model: str
    revision: str
    ambiguity_margin: float
    segments: list[WorkerSeparationInputSegmentsItem]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_separation_input_segments_item import WorkerSeparationInputSegmentsItem # noqa: PLC0415
        model = self.model

        revision = self.revision

        ambiguity_margin = self.ambiguity_margin

        segments = []
        for segments_item_data in self.segments:
            segments_item = segments_item_data.to_dict()
            segments.append(segments_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "model": model,
            "revision": revision,
            "ambiguity_margin": ambiguity_margin,
            "segments": segments,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_separation_input_segments_item import WorkerSeparationInputSegmentsItem # noqa: PLC0415
        d = dict(src_dict)
        model = d.pop("model")

        revision = d.pop("revision")

        ambiguity_margin = d.pop("ambiguity_margin")

        segments = []
        _segments = d.pop("segments")
        for segments_item_data in (_segments):
            segments_item = WorkerSeparationInputSegmentsItem.from_dict(segments_item_data)



            segments.append(segments_item)


        worker_separation_input = cls(
            model=model,
            revision=revision,
            ambiguity_margin=ambiguity_margin,
            segments=segments,
        )


        worker_separation_input.additional_properties = d
        return worker_separation_input

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
