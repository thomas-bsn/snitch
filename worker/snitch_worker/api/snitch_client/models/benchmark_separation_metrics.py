from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.benchmark_word_rates import BenchmarkWordRates





T = TypeVar("T", bound="BenchmarkSeparationMetrics")



@_attrs_define
class BenchmarkSeparationMetrics:
    """ Bout en bout, chaîne avec un modèle de séparation (#402, #403) : comptes des tâches de séparation, et cpWER et WER
    sans et avec séparation sur les mêmes clips, ceux dont la référence contient un Chevauchement.

        Attributes:
            tracks (int):
            zones (int):
            speakers_without_words_before (int): Locuteurs locaux sans aucun mot sur leur piste, règle d'attribution au mot
                de la production.
            speakers_without_words_after (int): Les mêmes, restés sans mot ajouté.
            sources (int):
            sources_without_speaker (int):
            words_added (int):
            words_added_absent_from_reference (int): Mots ajoutés absents des segments de référence qui recouvrent leur
                zone.
            words_duplicated_between_sources (int): Mots d'une source répétés, au même moment, dans une autre source de la
                même zone.
            min_clips (int): En dessous de ce nombre de clips avec Chevauchement, le rapport ne conclut pas.
            without_separation (BenchmarkWordRates):
            with_separation (BenchmarkWordRates):
     """

    tracks: int
    zones: int
    speakers_without_words_before: int
    speakers_without_words_after: int
    sources: int
    sources_without_speaker: int
    words_added: int
    words_added_absent_from_reference: int
    words_duplicated_between_sources: int
    min_clips: int
    without_separation: BenchmarkWordRates
    with_separation: BenchmarkWordRates
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.benchmark_word_rates import BenchmarkWordRates # noqa: PLC0415
        tracks = self.tracks

        zones = self.zones

        speakers_without_words_before = self.speakers_without_words_before

        speakers_without_words_after = self.speakers_without_words_after

        sources = self.sources

        sources_without_speaker = self.sources_without_speaker

        words_added = self.words_added

        words_added_absent_from_reference = self.words_added_absent_from_reference

        words_duplicated_between_sources = self.words_duplicated_between_sources

        min_clips = self.min_clips

        without_separation = self.without_separation.to_dict()

        with_separation = self.with_separation.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "tracks": tracks,
            "zones": zones,
            "speakers_without_words_before": speakers_without_words_before,
            "speakers_without_words_after": speakers_without_words_after,
            "sources": sources,
            "sources_without_speaker": sources_without_speaker,
            "words_added": words_added,
            "words_added_absent_from_reference": words_added_absent_from_reference,
            "words_duplicated_between_sources": words_duplicated_between_sources,
            "min_clips": min_clips,
            "without_separation": without_separation,
            "with_separation": with_separation,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.benchmark_word_rates import BenchmarkWordRates # noqa: PLC0415
        d = dict(src_dict)
        tracks = d.pop("tracks")

        zones = d.pop("zones")

        speakers_without_words_before = d.pop("speakers_without_words_before")

        speakers_without_words_after = d.pop("speakers_without_words_after")

        sources = d.pop("sources")

        sources_without_speaker = d.pop("sources_without_speaker")

        words_added = d.pop("words_added")

        words_added_absent_from_reference = d.pop("words_added_absent_from_reference")

        words_duplicated_between_sources = d.pop("words_duplicated_between_sources")

        min_clips = d.pop("min_clips")

        without_separation = BenchmarkWordRates.from_dict(d.pop("without_separation"))




        with_separation = BenchmarkWordRates.from_dict(d.pop("with_separation"))




        benchmark_separation_metrics = cls(
            tracks=tracks,
            zones=zones,
            speakers_without_words_before=speakers_without_words_before,
            speakers_without_words_after=speakers_without_words_after,
            sources=sources,
            sources_without_speaker=sources_without_speaker,
            words_added=words_added,
            words_added_absent_from_reference=words_added_absent_from_reference,
            words_duplicated_between_sources=words_duplicated_between_sources,
            min_clips=min_clips,
            without_separation=without_separation,
            with_separation=with_separation,
        )


        benchmark_separation_metrics.additional_properties = d
        return benchmark_separation_metrics

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
