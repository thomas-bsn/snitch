from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_voice_model_role import WorkerVoiceModelRole






T = TypeVar("T", bound="WorkerVoiceModel")



@_attrs_define
class WorkerVoiceModel:
    """
        Attributes:
            model_id (str):
            engine (str):
            repo (str):
            revision (str):
            role (WorkerVoiceModelRole):
            retry (bool):
     """

    model_id: str
    engine: str
    repo: str
    revision: str
    role: WorkerVoiceModelRole
    retry: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        model_id = self.model_id

        engine = self.engine

        repo = self.repo

        revision = self.revision

        role = self.role.value

        retry = self.retry


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "model_id": model_id,
            "engine": engine,
            "repo": repo,
            "revision": revision,
            "role": role,
            "retry": retry,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        model_id = d.pop("model_id")

        engine = d.pop("engine")

        repo = d.pop("repo")

        revision = d.pop("revision")

        role = WorkerVoiceModelRole(d.pop("role"))




        retry = d.pop("retry")

        worker_voice_model = cls(
            model_id=model_id,
            engine=engine,
            repo=repo,
            revision=revision,
            role=role,
            retry=retry,
        )


        worker_voice_model.additional_properties = d
        return worker_voice_model

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
