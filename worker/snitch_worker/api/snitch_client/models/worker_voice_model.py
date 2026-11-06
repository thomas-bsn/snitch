from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_voice_model_kind import WorkerVoiceModelKind
from ..models.worker_voice_model_role import WorkerVoiceModelRole
from ..types import UNSET, Unset






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
            kind (WorkerVoiceModelKind | Unset):
     """

    model_id: str
    engine: str
    repo: str
    revision: str
    role: WorkerVoiceModelRole
    retry: bool
    kind: WorkerVoiceModelKind | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        model_id = self.model_id

        engine = self.engine

        repo = self.repo

        revision = self.revision

        role = self.role.value

        retry = self.retry

        kind: str | Unset = UNSET
        if not isinstance(self.kind, Unset):
            kind = self.kind.value



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
        if kind is not UNSET:
            field_dict["kind"] = kind

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

        _kind = d.pop("kind", UNSET)
        kind: WorkerVoiceModelKind | Unset
        if isinstance(_kind,  Unset):
            kind = UNSET
        else:
            kind = WorkerVoiceModelKind(_kind)




        worker_voice_model = cls(
            model_id=model_id,
            engine=engine,
            repo=repo,
            revision=revision,
            role=role,
            retry=retry,
            kind=kind,
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
