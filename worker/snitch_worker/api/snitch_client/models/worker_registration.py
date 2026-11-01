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





T = TypeVar("T", bound="WorkerRegistration")



@_attrs_define
class WorkerRegistration:
    """
        Attributes:
            id (str):
            name (str):
            max_concurrency (int):
            reported_concurrency (int):
            last_seen_at (datetime.datetime | None):
            created_at (datetime.datetime):
            revoked_at (datetime.datetime | None):
            token (str):
            bundle (WorkerBundle): Compose et .env prêts à l'emploi, générés une seule fois avec le jeton ; jamais relus
                ensuite. Le .env contient le jeton.
            install_command (str | Unset): Commande d'installation Linux à copier une fois ; le code d'invitation éphémère
                n'est pas le jeton du Worker.
     """

    id: str
    name: str
    max_concurrency: int
    reported_concurrency: int
    last_seen_at: datetime.datetime | None
    created_at: datetime.datetime
    revoked_at: datetime.datetime | None
    token: str
    bundle: WorkerBundle
    install_command: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_bundle import WorkerBundle # noqa: PLC0415
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

        token = self.token

        bundle = self.bundle.to_dict()

        install_command = self.install_command


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
            "token": token,
            "bundle": bundle,
        })
        if install_command is not UNSET:
            field_dict["install_command"] = install_command

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_bundle import WorkerBundle # noqa: PLC0415
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


        token = d.pop("token")

        bundle = WorkerBundle.from_dict(d.pop("bundle"))




        install_command = d.pop("install_command", UNSET)

        worker_registration = cls(
            id=id,
            name=name,
            max_concurrency=max_concurrency,
            reported_concurrency=reported_concurrency,
            last_seen_at=last_seen_at,
            created_at=created_at,
            revoked_at=revoked_at,
            token=token,
            bundle=bundle,
            install_command=install_command,
        )


        worker_registration.additional_properties = d
        return worker_registration

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
