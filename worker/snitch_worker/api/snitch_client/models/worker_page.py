from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker import Worker
  from ..models.worker_page_transcription import WorkerPageTranscription





T = TypeVar("T", bound="WorkerPage")



@_attrs_define
class WorkerPage:
    """
        Attributes:
            first_worker (bool): Vrai tant qu'aucun worker n'a jamais été créé : le premier fixe le modèle et la langue de
                la bibliothèque.
            snitch_url (str): Adresse de Snitch telle que vue par cette requête, écrite dans les fichiers générés.
            transcription (WorkerPageTranscription): Profil de transcription actif de la bibliothèque.
            items (list[Worker]):
     """

    first_worker: bool
    snitch_url: str
    transcription: WorkerPageTranscription
    items: list[Worker]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker import Worker # noqa: PLC0415
        from ..models.worker_page_transcription import WorkerPageTranscription # noqa: PLC0415
        first_worker = self.first_worker

        snitch_url = self.snitch_url

        transcription = self.transcription.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "first_worker": first_worker,
            "snitch_url": snitch_url,
            "transcription": transcription,
            "items": items,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker import Worker # noqa: PLC0415
        from ..models.worker_page_transcription import WorkerPageTranscription # noqa: PLC0415
        d = dict(src_dict)
        first_worker = d.pop("first_worker")

        snitch_url = d.pop("snitch_url")

        transcription = WorkerPageTranscription.from_dict(d.pop("transcription"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = Worker.from_dict(items_item_data)



            items.append(items_item)


        worker_page = cls(
            first_worker=first_worker,
            snitch_url=snitch_url,
            transcription=transcription,
            items=items,
        )


        worker_page.additional_properties = d
        return worker_page

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
