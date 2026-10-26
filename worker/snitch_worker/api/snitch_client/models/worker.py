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
  from ..models.worker_machine import WorkerMachine
  from ..models.worker_models import WorkerModels





T = TypeVar("T", bound="Worker")



@_attrs_define
class Worker:
    """
        Attributes:
            id (str):
            name (str):
            max_concurrency (int):
            reported_concurrency (int):
            last_seen_at (datetime.datetime | None):
            created_at (datetime.datetime):
            revoked_at (datetime.datetime | None):
            connected (bool):
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
    connected: bool
    machine: WorkerMachine | Unset = UNSET
    remote_address: str | Unset = UNSET
    models: WorkerModels | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_machine import WorkerMachine # noqa: PLC0415
        from ..models.worker_models import WorkerModels # noqa: PLC0415
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

        connected = self.connected

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
            "connected": connected,
        })
        if machine is not UNSET:
            field_dict["machine"] = machine
        if remote_address is not UNSET:
            field_dict["remote_address"] = remote_address
        if models is not UNSET:
            field_dict["models"] = models

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_machine import WorkerMachine # noqa: PLC0415
        from ..models.worker_models import WorkerModels # noqa: PLC0415
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


        connected = d.pop("connected")

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




        worker = cls(
            id=id,
            name=name,
            max_concurrency=max_concurrency,
            reported_concurrency=reported_concurrency,
            last_seen_at=last_seen_at,
            created_at=created_at,
            revoked_at=revoked_at,
            connected=connected,
            machine=machine,
            remote_address=remote_address,
            models=models,
        )


        worker.additional_properties = d
        return worker

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
