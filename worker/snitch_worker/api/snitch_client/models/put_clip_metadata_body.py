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
            game_changed (bool | Unset):
            recorded_at_changed (bool | Unset):
            display_title (None | str | Unset): Titre affiché (200 caractères au plus, sans caractère de contrôle) ; vide ou
                absent pour le retirer. N'agit que si display_title_changed est vrai ; ne renomme jamais le fichier.
            display_title_changed (bool | Unset):
            game_reset (bool | Unset): Restaure le jeu détecté automatiquement ; 409 metadata_reset_unavailable s'il n'est
                pas disponible. Exclusif avec game_changed.
            recorded_at_reset (bool | Unset): Restaure la date détectée automatiquement ; 409 metadata_reset_unavailable
                s'il n'est pas disponible. Exclusif avec recorded_at_changed.
            game (None | str | Unset):
            recorded_at (datetime.datetime | None | Unset):
     """

    game_changed: bool | Unset = UNSET
    recorded_at_changed: bool | Unset = UNSET
    display_title: None | str | Unset = UNSET
    display_title_changed: bool | Unset = UNSET
    game_reset: bool | Unset = UNSET
    recorded_at_reset: bool | Unset = UNSET
    game: None | str | Unset = UNSET
    recorded_at: datetime.datetime | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        game_changed = self.game_changed

        recorded_at_changed = self.recorded_at_changed

        display_title: None | str | Unset
        if isinstance(self.display_title, Unset):
            display_title = UNSET
        else:
            display_title = self.display_title

        display_title_changed = self.display_title_changed

        game_reset = self.game_reset

        recorded_at_reset = self.recorded_at_reset

        game: None | str | Unset
        if isinstance(self.game, Unset):
            game = UNSET
        else:
            game = self.game

        recorded_at: None | str | Unset
        if isinstance(self.recorded_at, Unset):
            recorded_at = UNSET
        elif isinstance(self.recorded_at, datetime.datetime):
            recorded_at = self.recorded_at.isoformat()
        else:
            recorded_at = self.recorded_at


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
        })
        if game_changed is not UNSET:
            field_dict["game_changed"] = game_changed
        if recorded_at_changed is not UNSET:
            field_dict["recorded_at_changed"] = recorded_at_changed
        if display_title is not UNSET:
            field_dict["display_title"] = display_title
        if display_title_changed is not UNSET:
            field_dict["display_title_changed"] = display_title_changed
        if game_reset is not UNSET:
            field_dict["game_reset"] = game_reset
        if recorded_at_reset is not UNSET:
            field_dict["recorded_at_reset"] = recorded_at_reset
        if game is not UNSET:
            field_dict["game"] = game
        if recorded_at is not UNSET:
            field_dict["recorded_at"] = recorded_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        game_changed = d.pop("game_changed", UNSET)

        recorded_at_changed = d.pop("recorded_at_changed", UNSET)

        def _parse_display_title(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        display_title = _parse_display_title(d.pop("display_title", UNSET))


        display_title_changed = d.pop("display_title_changed", UNSET)

        game_reset = d.pop("game_reset", UNSET)

        recorded_at_reset = d.pop("recorded_at_reset", UNSET)

        def _parse_game(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        game = _parse_game(d.pop("game", UNSET))


        def _parse_recorded_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                recorded_at_type_0 = datetime.datetime.fromisoformat(data)



                return recorded_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        recorded_at = _parse_recorded_at(d.pop("recorded_at", UNSET))


        put_clip_metadata_body = cls(
            game_changed=game_changed,
            recorded_at_changed=recorded_at_changed,
            display_title=display_title,
            display_title_changed=display_title_changed,
            game_reset=game_reset,
            recorded_at_reset=recorded_at_reset,
            game=game,
            recorded_at=recorded_at,
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
