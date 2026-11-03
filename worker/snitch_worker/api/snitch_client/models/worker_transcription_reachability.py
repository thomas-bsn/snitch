from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.worker_transcription_failure_input import WorkerTranscriptionFailureInput





T = TypeVar("T", bound="WorkerTranscriptionReachability")



@_attrs_define
class WorkerTranscriptionReachability:
    """ Dernier test GET /v1/models par le Worker. Détail du dernier échec réservé aux administrateurs.

        Attributes:
            reachable (bool):
            checked_at (datetime.datetime):
            latency_ms (int):
            last_failure (None | WorkerTranscriptionFailureInput):
     """

    reachable: bool
    checked_at: datetime.datetime
    latency_ms: int
    last_failure: None | WorkerTranscriptionFailureInput





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_transcription_failure_input import WorkerTranscriptionFailureInput # noqa: PLC0415
        reachable = self.reachable

        checked_at = self.checked_at.isoformat()

        latency_ms = self.latency_ms

        last_failure: dict[str, Any] | None
        if isinstance(self.last_failure, WorkerTranscriptionFailureInput):
            last_failure = self.last_failure.to_dict()
        else:
            last_failure = self.last_failure


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "reachable": reachable,
            "checked_at": checked_at,
            "latency_ms": latency_ms,
            "last_failure": last_failure,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_transcription_failure_input import WorkerTranscriptionFailureInput # noqa: PLC0415
        d = dict(src_dict)
        reachable = d.pop("reachable")

        checked_at = datetime.datetime.fromisoformat(d.pop("checked_at"))




        latency_ms = d.pop("latency_ms")

        def _parse_last_failure(data: object) -> None | WorkerTranscriptionFailureInput:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                last_failure_type_1 = WorkerTranscriptionFailureInput.from_dict(data)



                return last_failure_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | WorkerTranscriptionFailureInput, data)

        last_failure = _parse_last_failure(d.pop("last_failure"))


        worker_transcription_reachability = cls(
            reachable=reachable,
            checked_at=checked_at,
            latency_ms=latency_ms,
            last_failure=last_failure,
        )

        return worker_transcription_reachability
