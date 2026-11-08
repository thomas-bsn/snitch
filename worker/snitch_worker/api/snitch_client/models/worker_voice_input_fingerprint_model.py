from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_voice_input_fingerprint_model_engine import WorkerVoiceInputFingerprintModelEngine






T = TypeVar("T", bound="WorkerVoiceInputFingerprintModel")



@_attrs_define
class WorkerVoiceInputFingerprintModel:
    """ Référence locale du modèle d’empreinte actif ou demandé, indépendante du moteur de diarisation. Absente uniquement
    pour les anciens serveurs et le fournisseur factice.

        Attributes:
            repo (str):
            revision (str):
            engine (WorkerVoiceInputFingerprintModelEngine):
     """

    repo: str
    revision: str
    engine: WorkerVoiceInputFingerprintModelEngine
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        repo = self.repo

        revision = self.revision

        engine = self.engine.value


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "repo": repo,
            "revision": revision,
            "engine": engine,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        repo = d.pop("repo")

        revision = d.pop("revision")

        engine = WorkerVoiceInputFingerprintModelEngine(d.pop("engine"))




        worker_voice_input_fingerprint_model = cls(
            repo=repo,
            revision=revision,
            engine=engine,
        )


        worker_voice_input_fingerprint_model.additional_properties = d
        return worker_voice_input_fingerprint_model

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
