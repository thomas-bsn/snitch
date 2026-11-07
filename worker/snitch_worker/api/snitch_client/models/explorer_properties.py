from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.explorer_properties_state import ExplorerPropertiesState
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.explorer_owner import ExplorerOwner





T = TypeVar("T", bound="ExplorerProperties")



@_attrs_define
class ExplorerProperties:
    """
        Attributes:
            path (str):
            name (str):
            directory (bool):
            size_bytes (int):
            video_count (int):
            folder_count (int):
            indexed_count (int):
            owners (list[ExplorerOwner]): Propriétaires des clips lisibles, du plus de vidéos au moins
            duration_seconds (float | None | Unset): Somme des durées des clips lisibles, nulle sans clip indexé
            added_at (datetime.datetime | None | Unset): Dossier : date du plus ancien clip indexé lisible
            recorded_at (datetime.datetime | None | Unset):
            game (str | Unset):
            state (ExplorerPropertiesState | Unset):
     """

    path: str
    name: str
    directory: bool
    size_bytes: int
    video_count: int
    folder_count: int
    indexed_count: int
    owners: list[ExplorerOwner]
    duration_seconds: float | None | Unset = UNSET
    added_at: datetime.datetime | None | Unset = UNSET
    recorded_at: datetime.datetime | None | Unset = UNSET
    game: str | Unset = UNSET
    state: ExplorerPropertiesState | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.explorer_owner import ExplorerOwner # noqa: PLC0415
        path = self.path

        name = self.name

        directory = self.directory

        size_bytes = self.size_bytes

        video_count = self.video_count

        folder_count = self.folder_count

        indexed_count = self.indexed_count

        owners = []
        for owners_item_data in self.owners:
            owners_item = owners_item_data.to_dict()
            owners.append(owners_item)



        duration_seconds: float | None | Unset
        if isinstance(self.duration_seconds, Unset):
            duration_seconds = UNSET
        else:
            duration_seconds = self.duration_seconds

        added_at: None | str | Unset
        if isinstance(self.added_at, Unset):
            added_at = UNSET
        elif isinstance(self.added_at, datetime.datetime):
            added_at = self.added_at.isoformat()
        else:
            added_at = self.added_at

        recorded_at: None | str | Unset
        if isinstance(self.recorded_at, Unset):
            recorded_at = UNSET
        elif isinstance(self.recorded_at, datetime.datetime):
            recorded_at = self.recorded_at.isoformat()
        else:
            recorded_at = self.recorded_at

        game = self.game

        state: str | Unset = UNSET
        if not isinstance(self.state, Unset):
            state = self.state.value



        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "path": path,
            "name": name,
            "directory": directory,
            "size_bytes": size_bytes,
            "video_count": video_count,
            "folder_count": folder_count,
            "indexed_count": indexed_count,
            "owners": owners,
        })
        if duration_seconds is not UNSET:
            field_dict["duration_seconds"] = duration_seconds
        if added_at is not UNSET:
            field_dict["added_at"] = added_at
        if recorded_at is not UNSET:
            field_dict["recorded_at"] = recorded_at
        if game is not UNSET:
            field_dict["game"] = game
        if state is not UNSET:
            field_dict["state"] = state

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.explorer_owner import ExplorerOwner # noqa: PLC0415
        d = dict(src_dict)
        path = d.pop("path")

        name = d.pop("name")

        directory = d.pop("directory")

        size_bytes = d.pop("size_bytes")

        video_count = d.pop("video_count")

        folder_count = d.pop("folder_count")

        indexed_count = d.pop("indexed_count")

        owners = []
        _owners = d.pop("owners")
        for owners_item_data in (_owners):
            owners_item = ExplorerOwner.from_dict(owners_item_data)



            owners.append(owners_item)


        def _parse_duration_seconds(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        duration_seconds = _parse_duration_seconds(d.pop("duration_seconds", UNSET))


        def _parse_added_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                added_at_type_0 = datetime.datetime.fromisoformat(data)



                return added_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        added_at = _parse_added_at(d.pop("added_at", UNSET))


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


        game = d.pop("game", UNSET)

        _state = d.pop("state", UNSET)
        state: ExplorerPropertiesState | Unset
        if isinstance(_state,  Unset):
            state = UNSET
        else:
            state = ExplorerPropertiesState(_state)




        explorer_properties = cls(
            path=path,
            name=name,
            directory=directory,
            size_bytes=size_bytes,
            video_count=video_count,
            folder_count=folder_count,
            indexed_count=indexed_count,
            owners=owners,
            duration_seconds=duration_seconds,
            added_at=added_at,
            recorded_at=recorded_at,
            game=game,
            state=state,
        )


        explorer_properties.additional_properties = d
        return explorer_properties

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
