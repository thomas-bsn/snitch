from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.put_worker_update_target_body_mode import PutWorkerUpdateTargetBodyMode
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="PutWorkerUpdateTargetBody")



@_attrs_define
class PutWorkerUpdateTargetBody:
    """
        Attributes:
            version (None | str):
            mode (PutWorkerUpdateTargetBodyMode | Unset):
     """

    version: None | str
    mode: PutWorkerUpdateTargetBodyMode | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        version: None | str
        version = self.version

        mode: str | Unset = UNSET
        if not isinstance(self.mode, Unset):
            mode = self.mode.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "version": version,
        })
        if mode is not UNSET:
            field_dict["mode"] = mode

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        def _parse_version(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        version = _parse_version(d.pop("version"))


        _mode = d.pop("mode", UNSET)
        mode: PutWorkerUpdateTargetBodyMode | Unset
        if isinstance(_mode,  Unset):
            mode = UNSET
        else:
            mode = PutWorkerUpdateTargetBodyMode(_mode)




        put_worker_update_target_body = cls(
            version=version,
            mode=mode,
        )

        return put_worker_update_target_body
