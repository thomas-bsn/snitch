from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_voice_model import WorkerVoiceModel





T = TypeVar("T", bound="WorkerVoiceModels")



@_attrs_define
class WorkerVoiceModels:
    """
        Attributes:
            models (list[WorkerVoiceModel]):
            remove (list[WorkerVoiceModel]):
     """

    models: list[WorkerVoiceModel]
    remove: list[WorkerVoiceModel]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_voice_model import WorkerVoiceModel # noqa: PLC0415
        models = []
        for models_item_data in self.models:
            models_item = models_item_data.to_dict()
            models.append(models_item)



        remove = []
        for remove_item_data in self.remove:
            remove_item = remove_item_data.to_dict()
            remove.append(remove_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "models": models,
            "remove": remove,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_voice_model import WorkerVoiceModel # noqa: PLC0415
        d = dict(src_dict)
        models = []
        _models = d.pop("models")
        for models_item_data in (_models):
            models_item = WorkerVoiceModel.from_dict(models_item_data)



            models.append(models_item)


        remove = []
        _remove = d.pop("remove")
        for remove_item_data in (_remove):
            remove_item = WorkerVoiceModel.from_dict(remove_item_data)



            remove.append(remove_item)


        worker_voice_models = cls(
            models=models,
            remove=remove,
        )


        worker_voice_models.additional_properties = d
        return worker_voice_models

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
