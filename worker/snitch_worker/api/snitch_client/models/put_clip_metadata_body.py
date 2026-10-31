from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PutClipMetadataBody")



@_attrs_define
class PutClipMetadataBody:
    """
        Attributes:
            game (None | str):
            recorded_at (datetime.datetime | None):
            game_changed (bool | Unset):
            recorded_at_changed (bool | Unset):
     """

    game: None | str
    recorded_at: datetime.datetime | None
    game_changed: bool | Unset = UNSET
    recorded_at_changed: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        game: None | str
        game = self.game

        recorded_at: None | str
        if isinstance(self.recorded_at, datetime.datetime):
            recorded_at = self.recorded_at.isoformat()
        else:
            recorded_at = self.recorded_at

        game_changed = self.game_changed

        recorded_at_changed = self.recorded_at_changed


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "game": game,
            "recorded_at": recorded_at,
        })
        if game_changed is not UNSET:
            field_dict["game_changed"] = game_changed
        if recorded_at_changed is not UNSET:
            field_dict["recorded_at_changed"] = recorded_at_changed

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        def _parse_game(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        game = _parse_game(d.pop("game"))


        def _parse_recorded_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                recorded_at_type_0 = datetime.datetime.fromisoformat(data)



                return recorded_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        recorded_at = _parse_recorded_at(d.pop("recorded_at"))


        game_changed = d.pop("game_changed", UNSET)

        recorded_at_changed = d.pop("recorded_at_changed", UNSET)

        put_clip_metadata_body = cls(
            game=game,
            recorded_at=recorded_at,
            game_changed=game_changed,
            recorded_at_changed=recorded_at_changed,
        )


        put_clip_metadata_body.additional_properties = d
        return put_clip_metadata_body

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
