from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ClipDetectedType0")



@_attrs_define
class ClipDetectedType0:
    """ Valeurs détectées automatiquement, conservées à la première modification manuelle pour pouvoir les restaurer ; une
    valeur n'est restaurable que si son indicateur *_available est vrai.

        Attributes:
            game_available (bool):
            game (None | str):
            game_origin (None | str):
            recorded_at_available (bool):
            recorded_at (datetime.datetime | None):
            recorded_at_origin (None | str):
     """

    game_available: bool
    game: None | str
    game_origin: None | str
    recorded_at_available: bool
    recorded_at: datetime.datetime | None
    recorded_at_origin: None | str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        game_available = self.game_available

        game: None | str
        game = self.game

        game_origin: None | str
        game_origin = self.game_origin

        recorded_at_available = self.recorded_at_available

        recorded_at: None | str
        if isinstance(self.recorded_at, datetime.datetime):
            recorded_at = self.recorded_at.isoformat()
        else:
            recorded_at = self.recorded_at

        recorded_at_origin: None | str
        recorded_at_origin = self.recorded_at_origin


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "game_available": game_available,
            "game": game,
            "game_origin": game_origin,
            "recorded_at_available": recorded_at_available,
            "recorded_at": recorded_at,
            "recorded_at_origin": recorded_at_origin,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        game_available = d.pop("game_available")

        def _parse_game(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        game = _parse_game(d.pop("game"))


        def _parse_game_origin(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        game_origin = _parse_game_origin(d.pop("game_origin"))


        recorded_at_available = d.pop("recorded_at_available")

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


        def _parse_recorded_at_origin(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        recorded_at_origin = _parse_recorded_at_origin(d.pop("recorded_at_origin"))


        clip_detected_type_0 = cls(
            game_available=game_available,
            game=game,
            game_origin=game_origin,
            recorded_at_available=recorded_at_available,
            recorded_at=recorded_at,
            recorded_at_origin=recorded_at_origin,
        )


        clip_detected_type_0.additional_properties = d
        return clip_detected_type_0

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
