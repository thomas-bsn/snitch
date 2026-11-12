from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_separation_input_segments_item import WorkerSeparationInputSegmentsItem
  from ..models.worker_separation_input_word_spans_item import WorkerSeparationInputWordSpansItem





T = TypeVar("T", bound="WorkerSeparationInput")



@_attrs_define
class WorkerSeparationInput:
    """ Tâche benchmark.separation.N (#402) : le modèle de séparation demandé et la transcription de la chaîne, dans le
    repère de la piste. Les tours sont dans voice.diarization, le modèle de transcription dans transcription_model.

        Attributes:
            silent_speakers (list[str]): Étiquettes des Locuteurs locaux de la piste sans aucun mot sous la règle
                d'attribution au mot de la production (#403) : le Worker n'a pas de règle propre.
            word_spans (list[WorkerSeparationInputWordSpansItem]): Temps des mots de chaque Locuteur local sous la même
                règle (#403) : le Worker n'ajoute les mots d'une source qu'à un Locuteur local sans mot dans la zone.
            model (str):
            revision (str):
            ambiguity_margin (float): Écart minimal de ressemblance entre les deux meilleurs Locuteurs locaux pour attribuer
                une Source séparée.
            segments (list[WorkerSeparationInputSegmentsItem]):
     """

    silent_speakers: list[str]
    word_spans: list[WorkerSeparationInputWordSpansItem]
    model: str
    revision: str
    ambiguity_margin: float
    segments: list[WorkerSeparationInputSegmentsItem]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_separation_input_segments_item import WorkerSeparationInputSegmentsItem # noqa: PLC0415
        from ..models.worker_separation_input_word_spans_item import WorkerSeparationInputWordSpansItem # noqa: PLC0415
        silent_speakers = self.silent_speakers



        word_spans = []
        for word_spans_item_data in self.word_spans:
            word_spans_item = word_spans_item_data.to_dict()
            word_spans.append(word_spans_item)



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
            "silent_speakers": silent_speakers,
            "word_spans": word_spans,
            "model": model,
            "revision": revision,
            "ambiguity_margin": ambiguity_margin,
            "segments": segments,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_separation_input_segments_item import WorkerSeparationInputSegmentsItem # noqa: PLC0415
        from ..models.worker_separation_input_word_spans_item import WorkerSeparationInputWordSpansItem # noqa: PLC0415
        d = dict(src_dict)
        silent_speakers = cast(list[str], d.pop("silent_speakers"))


        word_spans = []
        _word_spans = d.pop("word_spans")
        for word_spans_item_data in (_word_spans):
            word_spans_item = WorkerSeparationInputWordSpansItem.from_dict(word_spans_item_data)



            word_spans.append(word_spans_item)


        model = d.pop("model")

        revision = d.pop("revision")

        ambiguity_margin = d.pop("ambiguity_margin")

        segments = []
        _segments = d.pop("segments")
        for segments_item_data in (_segments):
            segments_item = WorkerSeparationInputSegmentsItem.from_dict(segments_item_data)



            segments.append(segments_item)


        worker_separation_input = cls(
            silent_speakers=silent_speakers,
            word_spans=word_spans,
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
