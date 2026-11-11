from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.root_scan_status import RootScanStatus
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.index_run_failure import IndexRunFailure
  from ..models.root_scan_excluded import RootScanExcluded





T = TypeVar("T", bound="RootScan")



@_attrs_define
class RootScan:
    """
        Attributes:
            id (int):
            status (RootScanStatus):
            created_at (datetime.datetime):
            finished_at (datetime.datetime | None):
            walking (bool): Vrai tant que le disque est parcouru.
            seen (int): Vidéos éligibles rencontrées jusqu'ici.
            queued (int): Vidéos mises en file par ce scan.
            done (int): Vidéos mises en file déjà prises par le scan géré.
            pending (int): Vidéos mises en file qui attendent encore.
            failed (int): Vidéos illisibles, disparues ou en erreur.
            up_to_date (int): Vidéos indexées dont le clip est déjà analysé.
            waiting_for_worker (bool): Des vidéos attendent et aucun Worker n'est en ligne.
            excluded (RootScanExcluded):
            unreadable_folders (int): Dossiers illisibles, sautés.
            unreadable_paths (list[str]):
            failures (list[IndexRunFailure]): Les premières vidéos en échec, avec leur code.
     """

    id: int
    status: RootScanStatus
    created_at: datetime.datetime
    finished_at: datetime.datetime | None
    walking: bool
    seen: int
    queued: int
    done: int
    pending: int
    failed: int
    up_to_date: int
    waiting_for_worker: bool
    excluded: RootScanExcluded
    unreadable_folders: int
    unreadable_paths: list[str]
    failures: list[IndexRunFailure]





    def to_dict(self) -> dict[str, Any]:
        from ..models.index_run_failure import IndexRunFailure # noqa: PLC0415
        from ..models.root_scan_excluded import RootScanExcluded # noqa: PLC0415
        id = self.id

        status = self.status.value

        created_at = self.created_at.isoformat()

        finished_at: None | str
        if isinstance(self.finished_at, datetime.datetime):
            finished_at = self.finished_at.isoformat()
        else:
            finished_at = self.finished_at

        walking = self.walking

        seen = self.seen

        queued = self.queued

        done = self.done

        pending = self.pending

        failed = self.failed

        up_to_date = self.up_to_date

        waiting_for_worker = self.waiting_for_worker

        excluded = self.excluded.to_dict()

        unreadable_folders = self.unreadable_folders

        unreadable_paths = self.unreadable_paths



        failures = []
        for failures_item_data in self.failures:
            failures_item = failures_item_data.to_dict()
            failures.append(failures_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "status": status,
            "created_at": created_at,
            "finished_at": finished_at,
            "walking": walking,
            "seen": seen,
            "queued": queued,
            "done": done,
            "pending": pending,
            "failed": failed,
            "up_to_date": up_to_date,
            "waiting_for_worker": waiting_for_worker,
            "excluded": excluded,
            "unreadable_folders": unreadable_folders,
            "unreadable_paths": unreadable_paths,
            "failures": failures,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.index_run_failure import IndexRunFailure # noqa: PLC0415
        from ..models.root_scan_excluded import RootScanExcluded # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        status = RootScanStatus(d.pop("status"))




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


        walking = d.pop("walking")

        seen = d.pop("seen")

        queued = d.pop("queued")

        done = d.pop("done")

        pending = d.pop("pending")

        failed = d.pop("failed")

        up_to_date = d.pop("up_to_date")

        waiting_for_worker = d.pop("waiting_for_worker")

        excluded = RootScanExcluded.from_dict(d.pop("excluded"))




        unreadable_folders = d.pop("unreadable_folders")

        unreadable_paths = cast(list[str], d.pop("unreadable_paths"))


        failures = []
        _failures = d.pop("failures")
        for failures_item_data in (_failures):
            failures_item = IndexRunFailure.from_dict(failures_item_data)



            failures.append(failures_item)


        root_scan = cls(
            id=id,
            status=status,
            created_at=created_at,
            finished_at=finished_at,
            walking=walking,
            seen=seen,
            queued=queued,
            done=done,
            pending=pending,
            failed=failed,
            up_to_date=up_to_date,
            waiting_for_worker=waiting_for_worker,
            excluded=excluded,
            unreadable_folders=unreadable_folders,
            unreadable_paths=unreadable_paths,
            failures=failures,
        )

        return root_scan
