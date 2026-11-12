from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="BenchmarkWordRates")



@_attrs_define
class BenchmarkWordRates:
    """
        Attributes:
            clips (int):
            reference_words (int):
            cp_word_errors (int):
            word_errors (int):
            cpwer (float | None):
            wer (float | None):
     """

    clips: int
    reference_words: int
    cp_word_errors: int
    word_errors: int
    cpwer: float | None
    wer: float | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        clips = self.clips

        reference_words = self.reference_words

        cp_word_errors = self.cp_word_errors

        word_errors = self.word_errors

        cpwer: float | None
        cpwer = self.cpwer

        wer: float | None
        wer = self.wer


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "clips": clips,
            "reference_words": reference_words,
            "cp_word_errors": cp_word_errors,
            "word_errors": word_errors,
            "cpwer": cpwer,
            "wer": wer,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clips = d.pop("clips")

        reference_words = d.pop("reference_words")

        cp_word_errors = d.pop("cp_word_errors")

        word_errors = d.pop("word_errors")

        def _parse_cpwer(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        cpwer = _parse_cpwer(d.pop("cpwer"))


        def _parse_wer(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        wer = _parse_wer(d.pop("wer"))


        benchmark_word_rates = cls(
            clips=clips,
            reference_words=reference_words,
            cp_word_errors=cp_word_errors,
            word_errors=word_errors,
            cpwer=cpwer,
            wer=wer,
        )


        benchmark_word_rates.additional_properties = d
        return benchmark_word_rates

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
