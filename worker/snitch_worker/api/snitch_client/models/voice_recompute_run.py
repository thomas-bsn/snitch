from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_recompute_run_status import VoiceRecomputeRunStatus
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.voice_recompute_report import VoiceRecomputeReport





T = TypeVar("T", bound="VoiceRecomputeRun")



@_attrs_define
class VoiceRecomputeRun:
    """
        Attributes:
            id (int):
            status (VoiceRecomputeRunStatus):
            announced (int): Nombre d'empreintes confirmé au lancement.
            author (None | str):
            created_at (datetime.datetime):
            finished_at (datetime.datetime | None):
            rematched_at (datetime.datetime | None): Fin du rapprochement de toute la bibliothèque relancé à la fin du
                recalcul.
            report (VoiceRecomputeReport):
     """

    id: int
    status: VoiceRecomputeRunStatus
    announced: int
    author: None | str
    created_at: datetime.datetime
    finished_at: datetime.datetime | None
    rematched_at: datetime.datetime | None
    report: VoiceRecomputeReport





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_recompute_report import VoiceRecomputeReport # noqa: PLC0415
        id = self.id

        status = self.status.value

        announced = self.announced

        author: None | str
        author = self.author

        created_at = self.created_at.isoformat()

        finished_at: None | str
        if isinstance(self.finished_at, datetime.datetime):
            finished_at = self.finished_at.isoformat()
        else:
            finished_at = self.finished_at

        rematched_at: None | str
        if isinstance(self.rematched_at, datetime.datetime):
            rematched_at = self.rematched_at.isoformat()
        else:
            rematched_at = self.rematched_at

        report = self.report.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "status": status,
            "announced": announced,
            "author": author,
            "created_at": created_at,
            "finished_at": finished_at,
            "rematched_at": rematched_at,
            "report": report,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_recompute_report import VoiceRecomputeReport # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        status = VoiceRecomputeRunStatus(d.pop("status"))




        announced = d.pop("announced")

        def _parse_author(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        author = _parse_author(d.pop("author"))


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


        def _parse_rematched_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                rematched_at_type_0 = datetime.datetime.fromisoformat(data)



                return rematched_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        rematched_at = _parse_rematched_at(d.pop("rematched_at"))


        report = VoiceRecomputeReport.from_dict(d.pop("report"))




        voice_recompute_run = cls(
            id=id,
            status=status,
            announced=announced,
            author=author,
            created_at=created_at,
            finished_at=finished_at,
            rematched_at=rematched_at,
            report=report,
        )

        return voice_recompute_run
