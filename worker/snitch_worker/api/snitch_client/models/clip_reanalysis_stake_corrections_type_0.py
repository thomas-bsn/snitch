from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ClipReanalysisStakeCorrectionsType0")



@_attrs_define
class ClipReanalysisStakeCorrectionsType0:
    """ Corrections de segments de la transcription visible (retranscription).

        Attributes:
            transcription_job_id (int):
            text_segments (int):
            speaker_segments (int):
            authors (list[str]):
            last_corrected_at (datetime.datetime | None):
     """

    transcription_job_id: int
    text_segments: int
    speaker_segments: int
    authors: list[str]
    last_corrected_at: datetime.datetime | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        transcription_job_id = self.transcription_job_id

        text_segments = self.text_segments

        speaker_segments = self.speaker_segments

        authors = self.authors



        last_corrected_at: None | str
        if isinstance(self.last_corrected_at, datetime.datetime):
            last_corrected_at = self.last_corrected_at.isoformat()
        else:
            last_corrected_at = self.last_corrected_at


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "transcription_job_id": transcription_job_id,
            "text_segments": text_segments,
            "speaker_segments": speaker_segments,
            "authors": authors,
            "last_corrected_at": last_corrected_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        transcription_job_id = d.pop("transcription_job_id")

        text_segments = d.pop("text_segments")

        speaker_segments = d.pop("speaker_segments")

        authors = cast(list[str], d.pop("authors"))


        def _parse_last_corrected_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_corrected_at_type_0 = datetime.datetime.fromisoformat(data)



                return last_corrected_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        last_corrected_at = _parse_last_corrected_at(d.pop("last_corrected_at"))


        clip_reanalysis_stake_corrections_type_0 = cls(
            transcription_job_id=transcription_job_id,
            text_segments=text_segments,
            speaker_segments=speaker_segments,
            authors=authors,
            last_corrected_at=last_corrected_at,
        )


        clip_reanalysis_stake_corrections_type_0.additional_properties = d
        return clip_reanalysis_stake_corrections_type_0

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
