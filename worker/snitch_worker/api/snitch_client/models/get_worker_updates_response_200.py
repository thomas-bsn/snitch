from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.get_worker_updates_response_200_mode import GetWorkerUpdatesResponse200Mode
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.get_worker_updates_response_200_workers_item import GetWorkerUpdatesResponse200WorkersItem





T = TypeVar("T", bound="GetWorkerUpdatesResponse200")



@_attrs_define
class GetWorkerUpdatesResponse200:
    """
        Attributes:
            mode (GetWorkerUpdatesResponse200Mode): Stratégie choisie. La version effective (target_version) est celle de
                l'administrateur en pinned, la plus haute version publiée déjà vue en latest.
            latest_version (None | str): Version stable la plus récente du catalogue, indépendamment du mode.
            checked_at (datetime.datetime | None): Dernière interrogation du registre par ce serveur (mémoire du processus),
                nulle tant qu'aucune n'a eu lieu.
            check_failed (bool): Vrai si la dernière interrogation du registre a échoué ; la dernière cible valide est
                conservée.
            target_version (None | str):
            target_image (None | str):
            image_repository (str):
            catalog (list[str]):
            catalog_available (bool):
            workers (list[GetWorkerUpdatesResponse200WorkersItem]):
     """

    mode: GetWorkerUpdatesResponse200Mode
    latest_version: None | str
    checked_at: datetime.datetime | None
    check_failed: bool
    target_version: None | str
    target_image: None | str
    image_repository: str
    catalog: list[str]
    catalog_available: bool
    workers: list[GetWorkerUpdatesResponse200WorkersItem]





    def to_dict(self) -> dict[str, Any]:
        from ..models.get_worker_updates_response_200_workers_item import GetWorkerUpdatesResponse200WorkersItem # noqa: PLC0415
        mode = self.mode.value

        latest_version: None | str
        latest_version = self.latest_version

        checked_at: None | str
        if isinstance(self.checked_at, datetime.datetime):
            checked_at = self.checked_at.isoformat()
        else:
            checked_at = self.checked_at

        check_failed = self.check_failed

        target_version: None | str
        target_version = self.target_version

        target_image: None | str
        target_image = self.target_image

        image_repository = self.image_repository

        catalog = self.catalog



        catalog_available = self.catalog_available

        workers = []
        for workers_item_data in self.workers:
            workers_item = workers_item_data.to_dict()
            workers.append(workers_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
            "latest_version": latest_version,
            "checked_at": checked_at,
            "check_failed": check_failed,
            "target_version": target_version,
            "target_image": target_image,
            "image_repository": image_repository,
            "catalog": catalog,
            "catalog_available": catalog_available,
            "workers": workers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.get_worker_updates_response_200_workers_item import GetWorkerUpdatesResponse200WorkersItem # noqa: PLC0415
        d = dict(src_dict)
        mode = GetWorkerUpdatesResponse200Mode(d.pop("mode"))




        def _parse_latest_version(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        latest_version = _parse_latest_version(d.pop("latest_version"))


        def _parse_checked_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                checked_at_type_0 = datetime.datetime.fromisoformat(data)



                return checked_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        checked_at = _parse_checked_at(d.pop("checked_at"))


        check_failed = d.pop("check_failed")

        def _parse_target_version(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        target_version = _parse_target_version(d.pop("target_version"))


        def _parse_target_image(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        target_image = _parse_target_image(d.pop("target_image"))


        image_repository = d.pop("image_repository")

        catalog = cast(list[str], d.pop("catalog"))


        catalog_available = d.pop("catalog_available")

        workers = []
        _workers = d.pop("workers")
        for workers_item_data in (_workers):
            workers_item = GetWorkerUpdatesResponse200WorkersItem.from_dict(workers_item_data)



            workers.append(workers_item)


        get_worker_updates_response_200 = cls(
            mode=mode,
            latest_version=latest_version,
            checked_at=checked_at,
            check_failed=check_failed,
            target_version=target_version,
            target_image=target_image,
            image_repository=image_repository,
            catalog=catalog,
            catalog_available=catalog_available,
            workers=workers,
        )

        return get_worker_updates_response_200
