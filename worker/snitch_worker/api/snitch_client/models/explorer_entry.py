from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.explorer_entry_state import ExplorerEntryState
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ExplorerEntry")



@_attrs_define
class ExplorerEntry:
    """
        Attributes:
            name (str):
            path (str): Chemin relatif à la Racine vidéo, ou clip:<id> pour un clip sans fichier dans la racine
            folder (str):
            directory (bool):
            size_bytes (int):
            private (bool):
            locked (bool):
            clip_id (int | None):
            owner_id (int | None):
            owner (str):
            duration_seconds (float | None):
            added_at (datetime.datetime | None):
            recorded_at (datetime.datetime | None):
            game (str):
            voices_pending (int):
            state (ExplorerEntryState | Unset):
            visibility (str | Unset):
     """

    name: str
    path: str
    folder: str
    directory: bool
    size_bytes: int
    private: bool
    locked: bool
    clip_id: int | None
    owner_id: int | None
    owner: str
    duration_seconds: float | None
    added_at: datetime.datetime | None
    recorded_at: datetime.datetime | None
    game: str
    voices_pending: int
    state: ExplorerEntryState | Unset = UNSET
    visibility: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        path = self.path

        folder = self.folder

        directory = self.directory

        size_bytes = self.size_bytes

        private = self.private

        locked = self.locked

        clip_id: int | None
        clip_id = self.clip_id

        owner_id: int | None
        owner_id = self.owner_id

        owner = self.owner

        duration_seconds: float | None
        duration_seconds = self.duration_seconds

        added_at: None | str
        if isinstance(self.added_at, datetime.datetime):
            added_at = self.added_at.isoformat()
        else:
            added_at = self.added_at

        recorded_at: None | str
        if isinstance(self.recorded_at, datetime.datetime):
            recorded_at = self.recorded_at.isoformat()
        else:
            recorded_at = self.recorded_at

        game = self.game

        voices_pending = self.voices_pending

        state: str | Unset = UNSET
        if not isinstance(self.state, Unset):
            state = self.state.value


        visibility = self.visibility


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "name": name,
            "path": path,
            "folder": folder,
            "directory": directory,
            "size_bytes": size_bytes,
            "private": private,
            "locked": locked,
            "clip_id": clip_id,
            "owner_id": owner_id,
            "owner": owner,
            "duration_seconds": duration_seconds,
            "added_at": added_at,
            "recorded_at": recorded_at,
            "game": game,
            "voices_pending": voices_pending,
        })
        if state is not UNSET:
            field_dict["state"] = state
        if visibility is not UNSET:
            field_dict["visibility"] = visibility

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        path = d.pop("path")

        folder = d.pop("folder")

        directory = d.pop("directory")

        size_bytes = d.pop("size_bytes")

        private = d.pop("private")

        locked = d.pop("locked")

        def _parse_clip_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        clip_id = _parse_clip_id(d.pop("clip_id"))


        def _parse_owner_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        owner_id = _parse_owner_id(d.pop("owner_id"))


        owner = d.pop("owner")

        def _parse_duration_seconds(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        duration_seconds = _parse_duration_seconds(d.pop("duration_seconds"))


        def _parse_added_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                added_at_type_0 = datetime.datetime.fromisoformat(data)



                return added_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        added_at = _parse_added_at(d.pop("added_at"))


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


        game = d.pop("game")

        voices_pending = d.pop("voices_pending")

        _state = d.pop("state", UNSET)
        state: ExplorerEntryState | Unset
        if isinstance(_state,  Unset):
            state = UNSET
        else:
            state = ExplorerEntryState(_state)




        visibility = d.pop("visibility", UNSET)

        explorer_entry = cls(
            name=name,
            path=path,
            folder=folder,
            directory=directory,
            size_bytes=size_bytes,
            private=private,
            locked=locked,
            clip_id=clip_id,
            owner_id=owner_id,
            owner=owner,
            duration_seconds=duration_seconds,
            added_at=added_at,
            recorded_at=recorded_at,
            game=game,
            voices_pending=voices_pending,
            state=state,
            visibility=visibility,
        )


        explorer_entry.additional_properties = d
        return explorer_entry

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
