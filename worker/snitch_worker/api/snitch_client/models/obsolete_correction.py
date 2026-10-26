from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.segment_history_entry import SegmentHistoryEntry





T = TypeVar("T", bound="ObsoleteCorrection")



@_attrs_define
class ObsoleteCorrection:
    """
        Attributes:
            transcription_job_id (int): Transcription remplacée à laquelle la correction appartenait.
            index (int): Rang du segment dans la transcription remplacée.
            original_text (str):
            text (None | str): Dernier texte corrigé ; null si le segment n'avait que des corrections de locuteur ou avait
                été restauré à l'original.
            obsolete_at (datetime.datetime):
            history (list[SegmentHistoryEntry]): Entrées immuables, de la plus récente à la plus ancienne.
     """

    transcription_job_id: int
    index: int
    original_text: str
    text: None | str
    obsolete_at: datetime.datetime
    history: list[SegmentHistoryEntry]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.segment_history_entry import SegmentHistoryEntry # noqa: PLC0415
        transcription_job_id = self.transcription_job_id

        index = self.index

        original_text = self.original_text

        text: None | str
        text = self.text

        obsolete_at = self.obsolete_at.isoformat()

        history = []
        for history_item_data in self.history:
            history_item = history_item_data.to_dict()
            history.append(history_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "transcription_job_id": transcription_job_id,
            "index": index,
            "original_text": original_text,
            "text": text,
            "obsolete_at": obsolete_at,
            "history": history,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.segment_history_entry import SegmentHistoryEntry # noqa: PLC0415
        d = dict(src_dict)
        transcription_job_id = d.pop("transcription_job_id")

        index = d.pop("index")

        original_text = d.pop("original_text")

        def _parse_text(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        text = _parse_text(d.pop("text"))


        obsolete_at = datetime.datetime.fromisoformat(d.pop("obsolete_at"))




        history = []
        _history = d.pop("history")
        for history_item_data in (_history):
            history_item = SegmentHistoryEntry.from_dict(history_item_data)



            history.append(history_item)


        obsolete_correction = cls(
            transcription_job_id=transcription_job_id,
            index=index,
            original_text=original_text,
            text=text,
            obsolete_at=obsolete_at,
            history=history,
        )


        obsolete_correction.additional_properties = d
        return obsolete_correction

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
