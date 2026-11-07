from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transcript_grouping_strategy import TranscriptGroupingStrategy
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transcript_source import TranscriptSource





T = TypeVar("T", bound="TranscriptGrouping")



@_attrs_define
class TranscriptGrouping:
    """ Preuve acoustique pour laquelle une source a été regroupée sous le passage conservé. La source reste consultable ;
    son texte n'est pas publié en double.

        Attributes:
            source (TranscriptSource):
            strategy (TranscriptGroupingStrategy): Règle acoustique activée qui a prouvé le doublon.
            strategy_version (str):
            lag_seconds (float): Décalage résiduel mesuré entre les deux signaux, en secondes.
            matched_start (float): Début, dans le repère du clip, de la plage prouvée identique côté passage conservé.
            matched_end (float):
            gain (float | Unset): Gain constant prouvé entre la source regroupée et le passage conservé (stratégie gain).
     """

    source: TranscriptSource
    strategy: TranscriptGroupingStrategy
    strategy_version: str
    lag_seconds: float
    matched_start: float
    matched_end: float
    gain: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transcript_source import TranscriptSource # noqa: PLC0415
        source = self.source.to_dict()

        strategy = self.strategy.value

        strategy_version = self.strategy_version

        lag_seconds = self.lag_seconds

        matched_start = self.matched_start

        matched_end = self.matched_end

        gain = self.gain


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "source": source,
            "strategy": strategy,
            "strategy_version": strategy_version,
            "lag_seconds": lag_seconds,
            "matched_start": matched_start,
            "matched_end": matched_end,
        })
        if gain is not UNSET:
            field_dict["gain"] = gain

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transcript_source import TranscriptSource # noqa: PLC0415
        d = dict(src_dict)
        source = TranscriptSource.from_dict(d.pop("source"))




        strategy = TranscriptGroupingStrategy(d.pop("strategy"))




        strategy_version = d.pop("strategy_version")

        lag_seconds = d.pop("lag_seconds")

        matched_start = d.pop("matched_start")

        matched_end = d.pop("matched_end")

        gain = d.pop("gain", UNSET)

        transcript_grouping = cls(
            source=source,
            strategy=strategy,
            strategy_version=strategy_version,
            lag_seconds=lag_seconds,
            matched_start=matched_start,
            matched_end=matched_end,
            gain=gain,
        )

        return transcript_grouping
