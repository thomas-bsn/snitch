from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.get_worker_updates_response_200_workers_item import GetWorkerUpdatesResponse200WorkersItem





T = TypeVar("T", bound="GetWorkerUpdatesResponse200")



@_attrs_define
class GetWorkerUpdatesResponse200:
    """
        Attributes:
            target_version (None | str):
            target_image (None | str):
            image_repository (str):
            catalog (list[str]):
            catalog_available (bool):
            workers (list[GetWorkerUpdatesResponse200WorkersItem]):
     """

    target_version: None | str
    target_image: None | str
    image_repository: str
    catalog: list[str]
    catalog_available: bool
    workers: list[GetWorkerUpdatesResponse200WorkersItem]





    def to_dict(self) -> dict[str, Any]:
        from ..models.get_worker_updates_response_200_workers_item import GetWorkerUpdatesResponse200WorkersItem # noqa: PLC0415
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
            target_version=target_version,
            target_image=target_image,
            image_repository=image_repository,
            catalog=catalog,
            catalog_available=catalog_available,
            workers=workers,
        )

        return get_worker_updates_response_200
