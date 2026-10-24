from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.transcript_word import TranscriptWord





T = TypeVar("T", bound="TranscriptSegment")



@_attrs_define
class TranscriptSegment:
    """
        Attributes:
            start (float):
            end (float):
            text (str):
            words (list[TranscriptWord]):
     """

    start: float
    end: float
    text: str
    words: list[TranscriptWord]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.transcript_word import TranscriptWord # noqa: PLC0415
        start = self.start

        end = self.end

        text = self.text

        words = []
        for words_item_data in self.words:
            words_item = words_item_data.to_dict()
            words.append(words_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "start": start,
            "end": end,
            "text": text,
            "words": words,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transcript_word import TranscriptWord # noqa: PLC0415
        d = dict(src_dict)
        start = d.pop("start")

        end = d.pop("end")

        text = d.pop("text")

        words = []
        _words = d.pop("words")
        for words_item_data in (_words):
            words_item = TranscriptWord.from_dict(words_item_data)



            words.append(words_item)


        transcript_segment = cls(
            start=start,
            end=end,
            text=text,
            words=words,
        )


        transcript_segment.additional_properties = d
        return transcript_segment

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
