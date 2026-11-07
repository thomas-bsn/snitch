from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_restart_command import WorkerRestartCommand





T = TypeVar("T", bound="PollWorkerRestartResponse200")



@_attrs_define
class PollWorkerRestartResponse200:
    """
        Attributes:
            command (None | WorkerRestartCommand):
     """

    command: None | WorkerRestartCommand
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_restart_command import WorkerRestartCommand # noqa: PLC0415
        command: dict[str, Any] | None
        if isinstance(self.command, WorkerRestartCommand):
            command = self.command.to_dict()
        else:
            command = self.command


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "command": command,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_restart_command import WorkerRestartCommand # noqa: PLC0415
        d = dict(src_dict)
        def _parse_command(data: object) -> None | WorkerRestartCommand:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                command_type_1 = WorkerRestartCommand.from_dict(data)



                return command_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | WorkerRestartCommand, data)

        command = _parse_command(d.pop("command"))


        poll_worker_restart_response_200 = cls(
            command=command,
        )


        poll_worker_restart_response_200.additional_properties = d
        return poll_worker_restart_response_200

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
