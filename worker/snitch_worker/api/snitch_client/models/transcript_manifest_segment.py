from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.transcript_manifest_word import TranscriptManifestWord
  from ..models.transcript_source import TranscriptSource





T = TypeVar("T", bound="TranscriptManifestSegment")



@_attrs_define
class TranscriptManifestSegment:
    """
        Attributes:
            index (int):
            start (float):
            end (float):
            sources (list[TranscriptSource]):
            words (list[TranscriptManifestWord]):
     """

    index: int
    start: float
    end: float
    sources: list[TranscriptSource]
    words: list[TranscriptManifestWord]





    def to_dict(self) -> dict[str, Any]:
        from ..models.transcript_manifest_word import TranscriptManifestWord # noqa: PLC0415
        from ..models.transcript_source import TranscriptSource # noqa: PLC0415
        index = self.index

        start = self.start

        end = self.end

        sources = []
        for sources_item_data in self.sources:
            sources_item = sources_item_data.to_dict()
            sources.append(sources_item)



        words = []
        for words_item_data in self.words:
            words_item = words_item_data.to_dict()
            words.append(words_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "index": index,
            "start": start,
            "end": end,
            "sources": sources,
            "words": words,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transcript_manifest_word import TranscriptManifestWord # noqa: PLC0415
        from ..models.transcript_source import TranscriptSource # noqa: PLC0415
        d = dict(src_dict)
        index = d.pop("index")

        start = d.pop("start")

        end = d.pop("end")

        sources = []
        _sources = d.pop("sources")
        for sources_item_data in (_sources):
            sources_item = TranscriptSource.from_dict(sources_item_data)



            sources.append(sources_item)


        words = []
        _words = d.pop("words")
        for words_item_data in (_words):
            words_item = TranscriptManifestWord.from_dict(words_item_data)



            words.append(words_item)


        transcript_manifest_segment = cls(
            index=index,
            start=start,
            end=end,
            sources=sources,
            words=words,
        )

        return transcript_manifest_segment
