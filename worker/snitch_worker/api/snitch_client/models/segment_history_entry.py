from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.segment_history_entry_kind import SegmentHistoryEntryKind
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.attribution_author import AttributionAuthor





T = TypeVar("T", bound="SegmentHistoryEntry")



@_attrs_define
class SegmentHistoryEntry:
    """
        Attributes:
            id (int):
            kind (SegmentHistoryEntryKind):
            old_text (None | str): Texte effectif avant l'écriture (texte d'origine s'il n'y avait pas de correction).
            new_text (None | str): Texte effectif après l'écriture.
            old_corrected (bool):
            new_corrected (bool): Faux si l'écriture a rétabli le texte d'origine.
            old_speaker_label (str):
            new_speaker_label (str):
            restored_from_event_id (int | None):
            restored_to_original (bool):
            reason (str):
            version_before (int):
            version_after (int):
            author (AttributionAuthor):
            created_at (datetime.datetime):
     """

    id: int
    kind: SegmentHistoryEntryKind
    old_text: None | str
    new_text: None | str
    old_corrected: bool
    new_corrected: bool
    old_speaker_label: str
    new_speaker_label: str
    restored_from_event_id: int | None
    restored_to_original: bool
    reason: str
    version_before: int
    version_after: int
    author: AttributionAuthor
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.attribution_author import AttributionAuthor # noqa: PLC0415
        id = self.id

        kind = self.kind.value

        old_text: None | str
        old_text = self.old_text

        new_text: None | str
        new_text = self.new_text

        old_corrected = self.old_corrected

        new_corrected = self.new_corrected

        old_speaker_label = self.old_speaker_label

        new_speaker_label = self.new_speaker_label

        restored_from_event_id: int | None
        restored_from_event_id = self.restored_from_event_id

        restored_to_original = self.restored_to_original

        reason = self.reason

        version_before = self.version_before

        version_after = self.version_after

        author = self.author.to_dict()

        created_at = self.created_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "kind": kind,
            "old_text": old_text,
            "new_text": new_text,
            "old_corrected": old_corrected,
            "new_corrected": new_corrected,
            "old_speaker_label": old_speaker_label,
            "new_speaker_label": new_speaker_label,
            "restored_from_event_id": restored_from_event_id,
            "restored_to_original": restored_to_original,
            "reason": reason,
            "version_before": version_before,
            "version_after": version_after,
            "author": author,
            "created_at": created_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.attribution_author import AttributionAuthor # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        kind = SegmentHistoryEntryKind(d.pop("kind"))




        def _parse_old_text(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        old_text = _parse_old_text(d.pop("old_text"))


        def _parse_new_text(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        new_text = _parse_new_text(d.pop("new_text"))


        old_corrected = d.pop("old_corrected")

        new_corrected = d.pop("new_corrected")

        old_speaker_label = d.pop("old_speaker_label")

        new_speaker_label = d.pop("new_speaker_label")

        def _parse_restored_from_event_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        restored_from_event_id = _parse_restored_from_event_id(d.pop("restored_from_event_id"))


        restored_to_original = d.pop("restored_to_original")

        reason = d.pop("reason")

        version_before = d.pop("version_before")

        version_after = d.pop("version_after")

        author = AttributionAuthor.from_dict(d.pop("author"))




        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        segment_history_entry = cls(
            id=id,
            kind=kind,
            old_text=old_text,
            new_text=new_text,
            old_corrected=old_corrected,
            new_corrected=new_corrected,
            old_speaker_label=old_speaker_label,
            new_speaker_label=new_speaker_label,
            restored_from_event_id=restored_from_event_id,
            restored_to_original=restored_to_original,
            reason=reason,
            version_before=version_before,
            version_after=version_after,
            author=author,
            created_at=created_at,
        )


        segment_history_entry.additional_properties = d
        return segment_history_entry

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
