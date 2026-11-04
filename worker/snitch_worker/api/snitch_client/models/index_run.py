from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.index_run_mode import IndexRunMode
from ..models.index_run_status import IndexRunStatus
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.index_run_failure import IndexRunFailure





T = TypeVar("T", bound="IndexRun")



@_attrs_define
class IndexRun:
    """
        Attributes:
            id (int):
            mode (IndexRunMode):
            status (IndexRunStatus):
            created_at (datetime.datetime):
            finished_at (datetime.datetime | None):
            total (int):
            indexed (int): Vidéos dont la fiche clip existe.
            failed (int): Vidéos illisibles ou disparues.
            pending (int):
            analysis_started (int): Clips de l'indexation dont une analyse a été mise en file (mode full).
            waiting_for_worker (bool): Mode full : des vidéos attendent et aucun Worker n'est en ligne.
            failures (list[IndexRunFailure]): Les premières vidéos en échec, avec leur code.
     """

    id: int
    mode: IndexRunMode
    status: IndexRunStatus
    created_at: datetime.datetime
    finished_at: datetime.datetime | None
    total: int
    indexed: int
    failed: int
    pending: int
    analysis_started: int
    waiting_for_worker: bool
    failures: list[IndexRunFailure]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.index_run_failure import IndexRunFailure # noqa: PLC0415
        id = self.id

        mode = self.mode.value

        status = self.status.value

        created_at = self.created_at.isoformat()

        finished_at: None | str
        if isinstance(self.finished_at, datetime.datetime):
            finished_at = self.finished_at.isoformat()
        else:
            finished_at = self.finished_at

        total = self.total

        indexed = self.indexed

        failed = self.failed

        pending = self.pending

        analysis_started = self.analysis_started

        waiting_for_worker = self.waiting_for_worker

        failures = []
        for failures_item_data in self.failures:
            failures_item = failures_item_data.to_dict()
            failures.append(failures_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "mode": mode,
            "status": status,
            "created_at": created_at,
            "finished_at": finished_at,
            "total": total,
            "indexed": indexed,
            "failed": failed,
            "pending": pending,
            "analysis_started": analysis_started,
            "waiting_for_worker": waiting_for_worker,
            "failures": failures,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.index_run_failure import IndexRunFailure # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        mode = IndexRunMode(d.pop("mode"))




        status = IndexRunStatus(d.pop("status"))




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


        total = d.pop("total")

        indexed = d.pop("indexed")

        failed = d.pop("failed")

        pending = d.pop("pending")

        analysis_started = d.pop("analysis_started")

        waiting_for_worker = d.pop("waiting_for_worker")

        failures = []
        _failures = d.pop("failures")
        for failures_item_data in (_failures):
            failures_item = IndexRunFailure.from_dict(failures_item_data)



            failures.append(failures_item)


        index_run = cls(
            id=id,
            mode=mode,
            status=status,
            created_at=created_at,
            finished_at=finished_at,
            total=total,
            indexed=indexed,
            failed=failed,
            pending=pending,
            analysis_started=analysis_started,
            waiting_for_worker=waiting_for_worker,
            failures=failures,
        )


        index_run.additional_properties = d
        return index_run

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
