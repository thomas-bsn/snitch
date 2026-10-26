from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="Setup")



@_attrs_define
class Setup:
    """
        Attributes:
            initialized (bool):
            discord_enabled (bool): True only when both the Discord application identifier and its authenticated encrypted
                secret are available.
            phase (int):
            deferred_steps (list[str]):
     """

    initialized: bool
    discord_enabled: bool
    phase: int
    deferred_steps: list[str]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        initialized = self.initialized

        discord_enabled = self.discord_enabled

        phase = self.phase

        deferred_steps = self.deferred_steps




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "initialized": initialized,
            "discord_enabled": discord_enabled,
            "phase": phase,
            "deferred_steps": deferred_steps,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        initialized = d.pop("initialized")

        discord_enabled = d.pop("discord_enabled")

        phase = d.pop("phase")

        deferred_steps = cast(list[str], d.pop("deferred_steps"))


        setup = cls(
            initialized=initialized,
            discord_enabled=discord_enabled,
            phase=phase,
            deferred_steps=deferred_steps,
        )


        setup.additional_properties = d
        return setup

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
