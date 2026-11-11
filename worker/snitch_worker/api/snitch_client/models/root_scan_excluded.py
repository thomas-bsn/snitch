from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="RootScanExcluded")



@_attrs_define
class RootScanExcluded:
    """
        Attributes:
            private_folders (int): Dossiers privés laissés de côté sans être ouverts.
            hidden (int): Dossiers et vidéos dont le nom commence par un point.
            system (int): Dossiers système du NAS (#recycle, @eaDir).
            private_clips (int): Vidéos indexées dont le clip est privé.
     """

    private_folders: int
    hidden: int
    system: int
    private_clips: int





    def to_dict(self) -> dict[str, Any]:
        private_folders = self.private_folders

        hidden = self.hidden

        system = self.system

        private_clips = self.private_clips


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "private_folders": private_folders,
            "hidden": hidden,
            "system": system,
            "private_clips": private_clips,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        private_folders = d.pop("private_folders")

        hidden = d.pop("hidden")

        system = d.pop("system")

        private_clips = d.pop("private_clips")

        root_scan_excluded = cls(
            private_folders=private_folders,
            hidden=hidden,
            system=system,
            private_clips=private_clips,
        )

        return root_scan_excluded
