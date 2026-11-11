from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.root_scan_excluded import RootScanExcluded





T = TypeVar("T", bound="RootScanPreview")



@_attrs_define
class RootScanPreview:
    """
        Attributes:
            seen (int): Vidéos éligibles rencontrées.
            to_queue (int): Vidéos qui seraient mises en file : non indexées, modifiées, ou indexées mais jamais analysées.
            up_to_date (int): Vidéos indexées dont le clip est déjà analysé : rien à faire.
            excluded (RootScanExcluded):
            unreadable_folders (int): Dossiers illisibles, sautés.
            unreadable_paths (list[str]): Les premiers dossiers illisibles, relatifs à la Racine.
     """

    seen: int
    to_queue: int
    up_to_date: int
    excluded: RootScanExcluded
    unreadable_folders: int
    unreadable_paths: list[str]





    def to_dict(self) -> dict[str, Any]:
        from ..models.root_scan_excluded import RootScanExcluded # noqa: PLC0415
        seen = self.seen

        to_queue = self.to_queue

        up_to_date = self.up_to_date

        excluded = self.excluded.to_dict()

        unreadable_folders = self.unreadable_folders

        unreadable_paths = self.unreadable_paths




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "seen": seen,
            "to_queue": to_queue,
            "up_to_date": up_to_date,
            "excluded": excluded,
            "unreadable_folders": unreadable_folders,
            "unreadable_paths": unreadable_paths,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.root_scan_excluded import RootScanExcluded # noqa: PLC0415
        d = dict(src_dict)
        seen = d.pop("seen")

        to_queue = d.pop("to_queue")

        up_to_date = d.pop("up_to_date")

        excluded = RootScanExcluded.from_dict(d.pop("excluded"))




        unreadable_folders = d.pop("unreadable_folders")

        unreadable_paths = cast(list[str], d.pop("unreadable_paths"))


        root_scan_preview = cls(
            seen=seen,
            to_queue=to_queue,
            up_to_date=up_to_date,
            excluded=excluded,
            unreadable_folders=unreadable_folders,
            unreadable_paths=unreadable_paths,
        )

        return root_scan_preview
