from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.engine_uninstall_preview_state import EngineUninstallPreviewState
from typing import cast

if TYPE_CHECKING:
  from ..models.engine_blockers import EngineBlockers





T = TypeVar("T", bound="EngineUninstallPreview")



@_attrs_define
class EngineUninstallPreview:
    """
        Attributes:
            state (EngineUninstallPreviewState):
            removable (bool):
            blockers (EngineBlockers):
     """

    state: EngineUninstallPreviewState
    removable: bool
    blockers: EngineBlockers
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.engine_blockers import EngineBlockers # noqa: PLC0415
        state = self.state.value

        removable = self.removable

        blockers = self.blockers.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "state": state,
            "removable": removable,
            "blockers": blockers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.engine_blockers import EngineBlockers # noqa: PLC0415
        d = dict(src_dict)
        state = EngineUninstallPreviewState(d.pop("state"))




        removable = d.pop("removable")

        blockers = EngineBlockers.from_dict(d.pop("blockers"))




        engine_uninstall_preview = cls(
            state=state,
            removable=removable,
            blockers=blockers,
        )


        engine_uninstall_preview.additional_properties = d
        return engine_uninstall_preview

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
