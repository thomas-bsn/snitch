from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.engine_recovery_result_status import EngineRecoveryResultStatus






T = TypeVar("T", bound="EngineRecoveryResult")



@_attrs_define
class EngineRecoveryResult:
    """
        Attributes:
            status (EngineRecoveryResultStatus):
     """

    status: EngineRecoveryResultStatus





    def to_dict(self) -> dict[str, Any]:
        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = EngineRecoveryResultStatus(d.pop("status"))




        engine_recovery_result = cls(
            status=status,
        )

        return engine_recovery_result
