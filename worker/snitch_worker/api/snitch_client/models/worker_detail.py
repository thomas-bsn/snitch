from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.worker_bundle import WorkerBundle
  from ..models.worker_install import WorkerInstall
  from ..models.worker_machine import WorkerMachine
  from ..models.worker_models import WorkerModels
  from ..models.worker_recent_task import WorkerRecentTask
  from ..models.worker_transcription_config import WorkerTranscriptionConfig





T = TypeVar("T", bound="WorkerDetail")



@_attrs_define
class WorkerDetail:
    """
        Attributes:
            id (str):
            name (str):
            max_concurrency (int):
            reported_concurrency (int):
            last_seen_at (datetime.datetime | None):
            created_at (datetime.datetime):
            revoked_at (datetime.datetime | None):
            revoke_reason (str): Motif facultatif de suppression, conservé dans l'audit minimal.
            revocation_acknowledged_at (datetime.datetime | None): Date de la première confirmation de la révocation par la
                machine du Worker (le Worker lui-même, qui efface alors le contenu de son volume de modèles et devient inactif,
                ou le lanceur, qui supprime le conteneur sans rien effacer). Absente : le nettoyage local reste en attente. Ne
                dit jamais que le volume Docker, le conteneur ou le dossier du lanceur sont supprimés.
            connected (bool):
            recent_tasks (list[WorkerRecentTask]): Dernières tâches de ce worker, une ligne par tâche, la plus récente
                d'abord.
            bundle (WorkerBundle): Compose et .env prêts à l'emploi, générés une seule fois avec le jeton ; jamais relus
                ensuite. Le .env contient le jeton.
            transcription_model (str | Unset): Nom facultatif du modèle propre au Worker.
            transcription_config (WorkerTranscriptionConfig | Unset): Statut de synchronisation de la configuration de
                transcription pour un Worker.
            transcription_config_ready (bool | Unset): Indique si le Worker supporte la configuration de transcription
                centralisée (capability transcription.config.v1).
            install (WorkerInstall | Unset): Progression d'installation, indépendante de la transcription
                (transcription_config) et de la voix (voice).
            machine (WorkerMachine | Unset): Informations non secrètes déclarées par le Worker ; le système est celui du
                conteneur.
            remote_address (str | Unset): Adresse IP de connexion observée par Snitch, sans port.
            models (WorkerModels | Unset): Modèles utilisés par un worker, par rôle. Tous les rôles sont facultatifs. Jamais
                de secret, de jeton ni de clé d’API : les valeurs sont bornées, limitées à des caractères sûrs, et les valeurs
                ressemblant à un secret sont refusées.
     """

    id: str
    name: str
    max_concurrency: int
    reported_concurrency: int
    last_seen_at: datetime.datetime | None
    created_at: datetime.datetime
    revoked_at: datetime.datetime | None
    revoke_reason: str
    revocation_acknowledged_at: datetime.datetime | None
    connected: bool
    recent_tasks: list[WorkerRecentTask]
    bundle: WorkerBundle
    transcription_model: str | Unset = UNSET
    transcription_config: WorkerTranscriptionConfig | Unset = UNSET
    transcription_config_ready: bool | Unset = UNSET
    install: WorkerInstall | Unset = UNSET
    machine: WorkerMachine | Unset = UNSET
    remote_address: str | Unset = UNSET
    models: WorkerModels | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_bundle import WorkerBundle # noqa: PLC0415
        from ..models.worker_install import WorkerInstall # noqa: PLC0415
        from ..models.worker_machine import WorkerMachine # noqa: PLC0415
        from ..models.worker_models import WorkerModels # noqa: PLC0415
        from ..models.worker_recent_task import WorkerRecentTask # noqa: PLC0415
        from ..models.worker_transcription_config import WorkerTranscriptionConfig # noqa: PLC0415
        id = self.id

        name = self.name

        max_concurrency = self.max_concurrency

        reported_concurrency = self.reported_concurrency

        last_seen_at: None | str
        if isinstance(self.last_seen_at, datetime.datetime):
            last_seen_at = self.last_seen_at.isoformat()
        else:
            last_seen_at = self.last_seen_at

        created_at = self.created_at.isoformat()

        revoked_at: None | str
        if isinstance(self.revoked_at, datetime.datetime):
            revoked_at = self.revoked_at.isoformat()
        else:
            revoked_at = self.revoked_at

        revoke_reason = self.revoke_reason

        revocation_acknowledged_at: None | str
        if isinstance(self.revocation_acknowledged_at, datetime.datetime):
            revocation_acknowledged_at = self.revocation_acknowledged_at.isoformat()
        else:
            revocation_acknowledged_at = self.revocation_acknowledged_at

        connected = self.connected

        recent_tasks = []
        for recent_tasks_item_data in self.recent_tasks:
            recent_tasks_item = recent_tasks_item_data.to_dict()
            recent_tasks.append(recent_tasks_item)



        bundle = self.bundle.to_dict()

        transcription_model = self.transcription_model

        transcription_config: dict[str, Any] | Unset = UNSET
        if not isinstance(self.transcription_config, Unset):
            transcription_config = self.transcription_config.to_dict()

        transcription_config_ready = self.transcription_config_ready

        install: dict[str, Any] | Unset = UNSET
        if not isinstance(self.install, Unset):
            install = self.install.to_dict()

        machine: dict[str, Any] | Unset = UNSET
        if not isinstance(self.machine, Unset):
            machine = self.machine.to_dict()

        remote_address = self.remote_address

        models: dict[str, Any] | Unset = UNSET
        if not isinstance(self.models, Unset):
            models = self.models.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "name": name,
            "max_concurrency": max_concurrency,
            "reported_concurrency": reported_concurrency,
            "last_seen_at": last_seen_at,
            "created_at": created_at,
            "revoked_at": revoked_at,
            "revoke_reason": revoke_reason,
            "revocation_acknowledged_at": revocation_acknowledged_at,
            "connected": connected,
            "recent_tasks": recent_tasks,
            "bundle": bundle,
        })
        if transcription_model is not UNSET:
            field_dict["transcription_model"] = transcription_model
        if transcription_config is not UNSET:
            field_dict["transcription_config"] = transcription_config
        if transcription_config_ready is not UNSET:
            field_dict["transcription_config_ready"] = transcription_config_ready
        if install is not UNSET:
            field_dict["install"] = install
        if machine is not UNSET:
            field_dict["machine"] = machine
        if remote_address is not UNSET:
            field_dict["remote_address"] = remote_address
        if models is not UNSET:
            field_dict["models"] = models

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_bundle import WorkerBundle # noqa: PLC0415
        from ..models.worker_install import WorkerInstall # noqa: PLC0415
        from ..models.worker_machine import WorkerMachine # noqa: PLC0415
        from ..models.worker_models import WorkerModels # noqa: PLC0415
        from ..models.worker_recent_task import WorkerRecentTask # noqa: PLC0415
        from ..models.worker_transcription_config import WorkerTranscriptionConfig # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        max_concurrency = d.pop("max_concurrency")

        reported_concurrency = d.pop("reported_concurrency")

        def _parse_last_seen_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_seen_at_type_0 = datetime.datetime.fromisoformat(data)



                return last_seen_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        last_seen_at = _parse_last_seen_at(d.pop("last_seen_at"))


        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        def _parse_revoked_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                revoked_at_type_0 = datetime.datetime.fromisoformat(data)



                return revoked_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        revoked_at = _parse_revoked_at(d.pop("revoked_at"))


        revoke_reason = d.pop("revoke_reason")

        def _parse_revocation_acknowledged_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                revocation_acknowledged_at_type_0 = datetime.datetime.fromisoformat(data)



                return revocation_acknowledged_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        revocation_acknowledged_at = _parse_revocation_acknowledged_at(d.pop("revocation_acknowledged_at"))


        connected = d.pop("connected")

        recent_tasks = []
        _recent_tasks = d.pop("recent_tasks")
        for recent_tasks_item_data in (_recent_tasks):
            recent_tasks_item = WorkerRecentTask.from_dict(recent_tasks_item_data)



            recent_tasks.append(recent_tasks_item)


        bundle = WorkerBundle.from_dict(d.pop("bundle"))




        transcription_model = d.pop("transcription_model", UNSET)

        _transcription_config = d.pop("transcription_config", UNSET)
        transcription_config: WorkerTranscriptionConfig | Unset
        if isinstance(_transcription_config,  Unset):
            transcription_config = UNSET
        else:
            transcription_config = WorkerTranscriptionConfig.from_dict(_transcription_config)




        transcription_config_ready = d.pop("transcription_config_ready", UNSET)

        _install = d.pop("install", UNSET)
        install: WorkerInstall | Unset
        if isinstance(_install,  Unset):
            install = UNSET
        else:
            install = WorkerInstall.from_dict(_install)




        _machine = d.pop("machine", UNSET)
        machine: WorkerMachine | Unset
        if isinstance(_machine,  Unset):
            machine = UNSET
        else:
            machine = WorkerMachine.from_dict(_machine)




        remote_address = d.pop("remote_address", UNSET)

        _models = d.pop("models", UNSET)
        models: WorkerModels | Unset
        if isinstance(_models,  Unset):
            models = UNSET
        else:
            models = WorkerModels.from_dict(_models)




        worker_detail = cls(
            id=id,
            name=name,
            max_concurrency=max_concurrency,
            reported_concurrency=reported_concurrency,
            last_seen_at=last_seen_at,
            created_at=created_at,
            revoked_at=revoked_at,
            revoke_reason=revoke_reason,
            revocation_acknowledged_at=revocation_acknowledged_at,
            connected=connected,
            recent_tasks=recent_tasks,
            bundle=bundle,
            transcription_model=transcription_model,
            transcription_config=transcription_config,
            transcription_config_ready=transcription_config_ready,
            install=install,
            machine=machine,
            remote_address=remote_address,
            models=models,
        )


        worker_detail.additional_properties = d
        return worker_detail

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
