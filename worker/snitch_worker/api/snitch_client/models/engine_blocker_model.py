from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="EngineBlockerModel")



@_attrs_define
class EngineBlockerModel:
    """
        Attributes:
            model_id (str):
            label (str):
            kind (str):
            state (str):
            busy (bool): Un job de ce Worker utilise encore le modèle.
            in_transition (bool): Le modèle est source ou cible d'une transition d'empreintes ouverte.
     """

    model_id: str
    label: str
    kind: str
    state: str
    busy: bool
    in_transition: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        model_id = self.model_id

        label = self.label

        kind = self.kind

        state = self.state

        busy = self.busy

        in_transition = self.in_transition


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "model_id": model_id,
            "label": label,
            "kind": kind,
            "state": state,
            "busy": busy,
            "in_transition": in_transition,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        model_id = d.pop("model_id")

        label = d.pop("label")

        kind = d.pop("kind")

        state = d.pop("state")

        busy = d.pop("busy")

        in_transition = d.pop("in_transition")

        engine_blocker_model = cls(
            model_id=model_id,
            label=label,
            kind=kind,
            state=state,
            busy=busy,
            in_transition=in_transition,
        )


        engine_blocker_model.additional_properties = d
        return engine_blocker_model

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
