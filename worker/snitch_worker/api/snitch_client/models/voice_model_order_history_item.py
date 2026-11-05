from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_model_order_history_item_action import VoiceModelOrderHistoryItemAction
from ..models.voice_model_order_history_item_kind import VoiceModelOrderHistoryItemKind
from ..models.voice_model_order_history_item_status import VoiceModelOrderHistoryItemStatus
from typing import cast
import datetime






T = TypeVar("T", bound="VoiceModelOrderHistoryItem")



@_attrs_define
class VoiceModelOrderHistoryItem:
    """
        Attributes:
            order_id (str):
            kind (VoiceModelOrderHistoryItemKind):
            action (VoiceModelOrderHistoryItemAction):
            model_id (str):
            label (str):
            repo (str):
            revision (str):
            status (VoiceModelOrderHistoryItemStatus):
            cause_code (str):
            author_name (str):
            created_at (datetime.datetime):
            finished_at (datetime.datetime | None):
     """

    order_id: str
    kind: VoiceModelOrderHistoryItemKind
    action: VoiceModelOrderHistoryItemAction
    model_id: str
    label: str
    repo: str
    revision: str
    status: VoiceModelOrderHistoryItemStatus
    cause_code: str
    author_name: str
    created_at: datetime.datetime
    finished_at: datetime.datetime | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        order_id = self.order_id

        kind = self.kind.value

        action = self.action.value

        model_id = self.model_id

        label = self.label

        repo = self.repo

        revision = self.revision

        status = self.status.value

        cause_code = self.cause_code

        author_name = self.author_name

        created_at = self.created_at.isoformat()

        finished_at: None | str
        if isinstance(self.finished_at, datetime.datetime):
            finished_at = self.finished_at.isoformat()
        else:
            finished_at = self.finished_at


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "order_id": order_id,
            "kind": kind,
            "action": action,
            "model_id": model_id,
            "label": label,
            "repo": repo,
            "revision": revision,
            "status": status,
            "cause_code": cause_code,
            "author_name": author_name,
            "created_at": created_at,
            "finished_at": finished_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        order_id = d.pop("order_id")

        kind = VoiceModelOrderHistoryItemKind(d.pop("kind"))




        action = VoiceModelOrderHistoryItemAction(d.pop("action"))




        model_id = d.pop("model_id")

        label = d.pop("label")

        repo = d.pop("repo")

        revision = d.pop("revision")

        status = VoiceModelOrderHistoryItemStatus(d.pop("status"))




        cause_code = d.pop("cause_code")

        author_name = d.pop("author_name")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        def _parse_finished_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                finished_at_type_0 = datetime.datetime.fromisoformat(data)



                return finished_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        finished_at = _parse_finished_at(d.pop("finished_at"))


        voice_model_order_history_item = cls(
            order_id=order_id,
            kind=kind,
            action=action,
            model_id=model_id,
            label=label,
            repo=repo,
            revision=revision,
            status=status,
            cause_code=cause_code,
            author_name=author_name,
            created_at=created_at,
            finished_at=finished_at,
        )


        voice_model_order_history_item.additional_properties = d
        return voice_model_order_history_item

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
