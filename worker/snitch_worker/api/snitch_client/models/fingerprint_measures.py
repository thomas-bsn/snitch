from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="FingerprintMeasures")



@_attrs_define
class FingerprintMeasures:
    """ Mesures de qualité d'une empreinte rapportées par le Worker (ADR 0064). Elles décrivent l'empreinte sans décider de
    son éligibilité.

        Attributes:
            version (int): Version des définitions du Worker (1 : intervalles de Parole nette d'au moins 0,5 s).
            net_speech_seconds (float): Parole nette entrée dans le vecteur ; 0 pour une empreinte chevauchée.
            overlap_ratio (float): Part de la parole du tour ou du Locuteur local en Chevauchement.
            overlapped (bool): Aucun intervalle net suffisant : vecteur calculé sur les tours entiers, jamais éligible comme
                référence.
            clipped_ratio (float | None): Part d'échantillons écrêtés de l'audio utilisé ; null si l'audio n'a pas été
                mesuré.
            mean_level_dbfs (float | None): Niveau moyen (RMS) de l'audio utilisé, en dBFS ; null si l'audio n'a pas été
                mesuré.
            coherence (float | None): Cosinus moyen entre les vecteurs des tours du Locuteur local ; null avec un seul tour
                et pour l'empreinte d'un tour.
     """

    version: int
    net_speech_seconds: float
    overlap_ratio: float
    overlapped: bool
    clipped_ratio: float | None
    mean_level_dbfs: float | None
    coherence: float | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        version = self.version

        net_speech_seconds = self.net_speech_seconds

        overlap_ratio = self.overlap_ratio

        overlapped = self.overlapped

        clipped_ratio: float | None
        clipped_ratio = self.clipped_ratio

        mean_level_dbfs: float | None
        mean_level_dbfs = self.mean_level_dbfs

        coherence: float | None
        coherence = self.coherence


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "version": version,
            "net_speech_seconds": net_speech_seconds,
            "overlap_ratio": overlap_ratio,
            "overlapped": overlapped,
            "clipped_ratio": clipped_ratio,
            "mean_level_dbfs": mean_level_dbfs,
            "coherence": coherence,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version")

        net_speech_seconds = d.pop("net_speech_seconds")

        overlap_ratio = d.pop("overlap_ratio")

        overlapped = d.pop("overlapped")

        def _parse_clipped_ratio(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        clipped_ratio = _parse_clipped_ratio(d.pop("clipped_ratio"))


        def _parse_mean_level_dbfs(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        mean_level_dbfs = _parse_mean_level_dbfs(d.pop("mean_level_dbfs"))


        def _parse_coherence(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        coherence = _parse_coherence(d.pop("coherence"))


        fingerprint_measures = cls(
            version=version,
            net_speech_seconds=net_speech_seconds,
            overlap_ratio=overlap_ratio,
            overlapped=overlapped,
            clipped_ratio=clipped_ratio,
            mean_level_dbfs=mean_level_dbfs,
            coherence=coherence,
        )


        fingerprint_measures.additional_properties = d
        return fingerprint_measures

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
