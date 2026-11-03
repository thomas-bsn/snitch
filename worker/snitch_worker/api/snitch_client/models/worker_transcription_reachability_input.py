from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.worker_transcription_failure_input import WorkerTranscriptionFailureInput





T = TypeVar("T", bound="WorkerTranscriptionReachabilityInput")



@_attrs_define
class WorkerTranscriptionReachabilityInput:
    """ Résultat de GET /v1/models sur le serveur de transcription. Jamais d'identifiants : l'URL est déjà sans utilisateur
    ni requête.

        Attributes:
            checked_at (datetime.datetime):
            reachable (bool):
            latency_ms (int):
            last_failure (WorkerTranscriptionFailureInput | Unset):
     """

    checked_at: datetime.datetime
    reachable: bool
    latency_ms: int
    last_failure: WorkerTranscriptionFailureInput | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_transcription_failure_input import WorkerTranscriptionFailureInput # noqa: PLC0415
        checked_at = self.checked_at.isoformat()

        reachable = self.reachable

        latency_ms = self.latency_ms

        last_failure: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last_failure, Unset):
            last_failure = self.last_failure.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checked_at": checked_at,
            "reachable": reachable,
            "latency_ms": latency_ms,
        })
        if last_failure is not UNSET:
            field_dict["last_failure"] = last_failure

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_transcription_failure_input import WorkerTranscriptionFailureInput # noqa: PLC0415
        d = dict(src_dict)
        checked_at = datetime.datetime.fromisoformat(d.pop("checked_at"))




        reachable = d.pop("reachable")

        latency_ms = d.pop("latency_ms")

        _last_failure = d.pop("last_failure", UNSET)
        last_failure: WorkerTranscriptionFailureInput | Unset
        if isinstance(_last_failure,  Unset):
            last_failure = UNSET
        else:
            last_failure = WorkerTranscriptionFailureInput.from_dict(_last_failure)




        worker_transcription_reachability_input = cls(
            checked_at=checked_at,
            reachable=reachable,
            latency_ms=latency_ms,
            last_failure=last_failure,
        )

        return worker_transcription_reachability_input
