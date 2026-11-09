from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_model_order_preview_state import VoiceModelOrderPreviewState
from ..types import UNSET, Unset






T = TypeVar("T", bound="VoiceModelOrderPreview")



@_attrs_define
class VoiceModelOrderPreview:
    """
        Attributes:
            model_id (str):
            engine_required (bool): Vrai quand le modèle demande un engine à la demande ; faux pour un moteur embarqué ou un
                adaptateur signé.
            disk_bytes (int): Disque que l'engine va occuper ; 0 s'il est déjà prêt.
            estimated_seconds (int): Durée estimée de l'installation ; 0 s'il est déjà prêt.
            active (bool): Faux quand la famille s'installe pour cette tâche mais qu'aucun adaptateur ne l'exécute
                (séparation SpeechBrain) : l'engine est commandé avec le modèle, le modèle reste indisponible.
            family (str | Unset):
            engine (str | Unset):
            version (str | Unset):
            kind (str | Unset): Tâche de l'engine (celle du modèle).
            state (VoiceModelOrderPreviewState | Unset):
            blocker (str | Unset): Cause stable qui refuserait l'ordre d'engine sur ce Worker (matériel), absente sinon.
     """

    model_id: str
    engine_required: bool
    disk_bytes: int
    estimated_seconds: int
    active: bool
    family: str | Unset = UNSET
    engine: str | Unset = UNSET
    version: str | Unset = UNSET
    kind: str | Unset = UNSET
    state: VoiceModelOrderPreviewState | Unset = UNSET
    blocker: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        model_id = self.model_id

        engine_required = self.engine_required

        disk_bytes = self.disk_bytes

        estimated_seconds = self.estimated_seconds

        active = self.active

        family = self.family

        engine = self.engine

        version = self.version

        kind = self.kind

        state: str | Unset = UNSET
        if not isinstance(self.state, Unset):
            state = self.state.value


        blocker = self.blocker


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "model_id": model_id,
            "engine_required": engine_required,
            "disk_bytes": disk_bytes,
            "estimated_seconds": estimated_seconds,
            "active": active,
        })
        if family is not UNSET:
            field_dict["family"] = family
        if engine is not UNSET:
            field_dict["engine"] = engine
        if version is not UNSET:
            field_dict["version"] = version
        if kind is not UNSET:
            field_dict["kind"] = kind
        if state is not UNSET:
            field_dict["state"] = state
        if blocker is not UNSET:
            field_dict["blocker"] = blocker

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        model_id = d.pop("model_id")

        engine_required = d.pop("engine_required")

        disk_bytes = d.pop("disk_bytes")

        estimated_seconds = d.pop("estimated_seconds")

        active = d.pop("active")

        family = d.pop("family", UNSET)

        engine = d.pop("engine", UNSET)

        version = d.pop("version", UNSET)

        kind = d.pop("kind", UNSET)

        _state = d.pop("state", UNSET)
        state: VoiceModelOrderPreviewState | Unset
        if isinstance(_state,  Unset):
            state = UNSET
        else:
            state = VoiceModelOrderPreviewState(_state)




        blocker = d.pop("blocker", UNSET)

        voice_model_order_preview = cls(
            model_id=model_id,
            engine_required=engine_required,
            disk_bytes=disk_bytes,
            estimated_seconds=estimated_seconds,
            active=active,
            family=family,
            engine=engine,
            version=version,
            kind=kind,
            state=state,
            blocker=blocker,
        )

        return voice_model_order_preview
