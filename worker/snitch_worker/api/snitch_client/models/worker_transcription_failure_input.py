from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_transcription_failure_input_error_type import WorkerTranscriptionFailureInputErrorType
from ..models.worker_transcription_failure_input_method import WorkerTranscriptionFailureInputMethod
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WorkerTranscriptionFailureInput")



@_attrs_define
class WorkerTranscriptionFailureInput:
    """
        Attributes:
            failed_at (datetime.datetime):
            method (WorkerTranscriptionFailureInputMethod):
            url (str):
            latency_ms (int):
            error_type (WorkerTranscriptionFailureInputErrorType):
            body_excerpt (str): Extrait borné de la réponse, vide hors statut HTTP ou corps invalide.
            status_code (int | None | Unset):
     """

    failed_at: datetime.datetime
    method: WorkerTranscriptionFailureInputMethod
    url: str
    latency_ms: int
    error_type: WorkerTranscriptionFailureInputErrorType
    body_excerpt: str
    status_code: int | None | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        failed_at = self.failed_at.isoformat()

        method = self.method.value

        url = self.url

        latency_ms = self.latency_ms

        error_type = self.error_type.value

        body_excerpt = self.body_excerpt

        status_code: int | None | Unset
        if isinstance(self.status_code, Unset):
            status_code = UNSET
        else:
            status_code = self.status_code


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "failed_at": failed_at,
            "method": method,
            "url": url,
            "latency_ms": latency_ms,
            "error_type": error_type,
            "body_excerpt": body_excerpt,
        })
        if status_code is not UNSET:
            field_dict["status_code"] = status_code

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        failed_at = datetime.datetime.fromisoformat(d.pop("failed_at"))




        method = WorkerTranscriptionFailureInputMethod(d.pop("method"))




        url = d.pop("url")

        latency_ms = d.pop("latency_ms")

        error_type = WorkerTranscriptionFailureInputErrorType(d.pop("error_type"))




        body_excerpt = d.pop("body_excerpt")

        def _parse_status_code(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        status_code = _parse_status_code(d.pop("status_code", UNSET))


        worker_transcription_failure_input = cls(
            failed_at=failed_at,
            method=method,
            url=url,
            latency_ms=latency_ms,
            error_type=error_type,
            body_excerpt=body_excerpt,
            status_code=status_code,
        )

        return worker_transcription_failure_input
