from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="PeopleToProcessClip")



@_attrs_define
class PeopleToProcessClip:
    """
        Attributes:
            clip_id (int):
            clip_name (str):
            to_confirm (int):
            ambiguous (int):
            unknown (int):
            groups (int):
            folder (str): Dossier du clip dans sa source (chemin relatif sans le nom du fichier), vide s'il n'en a pas.
            recorded_at (datetime.datetime):
            duration_seconds (float | None):
            analysed_recently (bool): Une analyse du clip s'est terminée dans les 24 dernières heures.
     """

    clip_id: int
    clip_name: str
    to_confirm: int
    ambiguous: int
    unknown: int
    groups: int
    folder: str
    recorded_at: datetime.datetime
    duration_seconds: float | None
    analysed_recently: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        clip_id = self.clip_id

        clip_name = self.clip_name

        to_confirm = self.to_confirm

        ambiguous = self.ambiguous

        unknown = self.unknown

        groups = self.groups

        folder = self.folder

        recorded_at = self.recorded_at.isoformat()

        duration_seconds: float | None
        duration_seconds = self.duration_seconds

        analysed_recently = self.analysed_recently


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "clip_id": clip_id,
            "clip_name": clip_name,
            "to_confirm": to_confirm,
            "ambiguous": ambiguous,
            "unknown": unknown,
            "groups": groups,
            "folder": folder,
            "recorded_at": recorded_at,
            "duration_seconds": duration_seconds,
            "analysed_recently": analysed_recently,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clip_id = d.pop("clip_id")

        clip_name = d.pop("clip_name")

        to_confirm = d.pop("to_confirm")

        ambiguous = d.pop("ambiguous")

        unknown = d.pop("unknown")

        groups = d.pop("groups")

        folder = d.pop("folder")

        recorded_at = datetime.datetime.fromisoformat(d.pop("recorded_at"))




        def _parse_duration_seconds(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        duration_seconds = _parse_duration_seconds(d.pop("duration_seconds"))


        analysed_recently = d.pop("analysed_recently")

        people_to_process_clip = cls(
            clip_id=clip_id,
            clip_name=clip_name,
            to_confirm=to_confirm,
            ambiguous=ambiguous,
            unknown=unknown,
            groups=groups,
            folder=folder,
            recorded_at=recorded_at,
            duration_seconds=duration_seconds,
            analysed_recently=analysed_recently,
        )


        people_to_process_clip.additional_properties = d
        return people_to_process_clip

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
