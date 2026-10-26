from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.attribution_author import AttributionAuthor





T = TypeVar("T", bound="SegmentCorrection")



@_attrs_define
class SegmentCorrection:
    """
        Attributes:
            transcription_job_id (int):
            index (int):
            version (int): 0 tant que le segment n'a jamais été écrit ; incrémentée à chaque écriture.
            text (str): Texte effectif : le texte corrigé s'il existe, sinon le texte d'origine.
            original_text (str):
            corrected (bool):
            author (AttributionAuthor | None):
            corrected_at (datetime.datetime | None):
     """

    transcription_job_id: int
    index: int
    version: int
    text: str
    original_text: str
    corrected: bool
    author: AttributionAuthor | None
    corrected_at: datetime.datetime | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.attribution_author import AttributionAuthor # noqa: PLC0415
        transcription_job_id = self.transcription_job_id

        index = self.index

        version = self.version

        text = self.text

        original_text = self.original_text

        corrected = self.corrected

        author: dict[str, Any] | None
        if isinstance(self.author, AttributionAuthor):
            author = self.author.to_dict()
        else:
            author = self.author

        corrected_at: None | str
        if isinstance(self.corrected_at, datetime.datetime):
            corrected_at = self.corrected_at.isoformat()
        else:
            corrected_at = self.corrected_at


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "transcription_job_id": transcription_job_id,
            "index": index,
            "version": version,
            "text": text,
            "original_text": original_text,
            "corrected": corrected,
            "author": author,
            "corrected_at": corrected_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.attribution_author import AttributionAuthor # noqa: PLC0415
        d = dict(src_dict)
        transcription_job_id = d.pop("transcription_job_id")

        index = d.pop("index")

        version = d.pop("version")

        text = d.pop("text")

        original_text = d.pop("original_text")

        corrected = d.pop("corrected")

        def _parse_author(data: object) -> AttributionAuthor | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                author_type_1 = AttributionAuthor.from_dict(data)



                return author_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(AttributionAuthor | None, data)

        author = _parse_author(d.pop("author"))


        def _parse_corrected_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                corrected_at_type_0 = datetime.datetime.fromisoformat(data)



                return corrected_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        corrected_at = _parse_corrected_at(d.pop("corrected_at"))


        segment_correction = cls(
            transcription_job_id=transcription_job_id,
            index=index,
            version=version,
            text=text,
            original_text=original_text,
            corrected=corrected,
            author=author,
            corrected_at=corrected_at,
        )


        segment_correction.additional_properties = d
        return segment_correction

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
