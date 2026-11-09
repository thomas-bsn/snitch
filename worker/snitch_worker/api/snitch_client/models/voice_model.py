from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_model_kind import VoiceModelKind
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.voice_model_worker_state import VoiceModelWorkerState





T = TypeVar("T", bound="VoiceModel")



@_attrs_define
class VoiceModel:
    """
        Attributes:
            id (str):
            kind (VoiceModelKind):
            label (str):
            vram_bytes (int): Mémoire GPU nécessaire, 0 si inconnue.
            enabled (bool): Une référence désactivée ne peut plus être choisie ; son historique reste lisible.
            supported (bool): Faux quand aucun adaptateur approuvé ne sait l'exécuter (toujours le cas de la séparation).
            unsupported_reason (str):
            engine (str):
            repo (str):
            revision (str):
            license_ (str):
            gated (bool):
            max_speakers (int | None):
            size_bytes (int):
            supports_num_speakers (bool):
            active (bool):
            candidate (bool):
            installed (bool): Au moins un Worker a le modèle prêt.
            workers (list[VoiceModelWorkerState]):
            family (str | Unset): Famille d'engine à la demande dont le modèle a besoin (ADR 0056) ; absente pour un moteur
                embarqué ou un adaptateur signé.
     """

    id: str
    kind: VoiceModelKind
    label: str
    vram_bytes: int
    enabled: bool
    supported: bool
    unsupported_reason: str
    engine: str
    repo: str
    revision: str
    license_: str
    gated: bool
    max_speakers: int | None
    size_bytes: int
    supports_num_speakers: bool
    active: bool
    candidate: bool
    installed: bool
    workers: list[VoiceModelWorkerState]
    family: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_model_worker_state import VoiceModelWorkerState # noqa: PLC0415
        id = self.id

        kind = self.kind.value

        label = self.label

        vram_bytes = self.vram_bytes

        enabled = self.enabled

        supported = self.supported

        unsupported_reason = self.unsupported_reason

        engine = self.engine

        repo = self.repo

        revision = self.revision

        license_ = self.license_

        gated = self.gated

        max_speakers: int | None
        max_speakers = self.max_speakers

        size_bytes = self.size_bytes

        supports_num_speakers = self.supports_num_speakers

        active = self.active

        candidate = self.candidate

        installed = self.installed

        workers = []
        for workers_item_data in self.workers:
            workers_item = workers_item_data.to_dict()
            workers.append(workers_item)



        family = self.family


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "kind": kind,
            "label": label,
            "vram_bytes": vram_bytes,
            "enabled": enabled,
            "supported": supported,
            "unsupported_reason": unsupported_reason,
            "engine": engine,
            "repo": repo,
            "revision": revision,
            "license": license_,
            "gated": gated,
            "max_speakers": max_speakers,
            "size_bytes": size_bytes,
            "supports_num_speakers": supports_num_speakers,
            "active": active,
            "candidate": candidate,
            "installed": installed,
            "workers": workers,
        })
        if family is not UNSET:
            field_dict["family"] = family

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_model_worker_state import VoiceModelWorkerState # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        kind = VoiceModelKind(d.pop("kind"))




        label = d.pop("label")

        vram_bytes = d.pop("vram_bytes")

        enabled = d.pop("enabled")

        supported = d.pop("supported")

        unsupported_reason = d.pop("unsupported_reason")

        engine = d.pop("engine")

        repo = d.pop("repo")

        revision = d.pop("revision")

        license_ = d.pop("license")

        gated = d.pop("gated")

        def _parse_max_speakers(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        max_speakers = _parse_max_speakers(d.pop("max_speakers"))


        size_bytes = d.pop("size_bytes")

        supports_num_speakers = d.pop("supports_num_speakers")

        active = d.pop("active")

        candidate = d.pop("candidate")

        installed = d.pop("installed")

        workers = []
        _workers = d.pop("workers")
        for workers_item_data in (_workers):
            workers_item = VoiceModelWorkerState.from_dict(workers_item_data)



            workers.append(workers_item)


        family = d.pop("family", UNSET)

        voice_model = cls(
            id=id,
            kind=kind,
            label=label,
            vram_bytes=vram_bytes,
            enabled=enabled,
            supported=supported,
            unsupported_reason=unsupported_reason,
            engine=engine,
            repo=repo,
            revision=revision,
            license_=license_,
            gated=gated,
            max_speakers=max_speakers,
            size_bytes=size_bytes,
            supports_num_speakers=supports_num_speakers,
            active=active,
            candidate=candidate,
            installed=installed,
            workers=workers,
            family=family,
        )


        voice_model.additional_properties = d
        return voice_model

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
