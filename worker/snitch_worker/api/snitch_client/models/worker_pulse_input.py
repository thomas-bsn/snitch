from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.worker_machine import WorkerMachine
  from ..models.worker_models import WorkerModels





T = TypeVar("T", bound="WorkerPulseInput")



@_attrs_define
class WorkerPulseInput:
    """
        Attributes:
            capacity (int):
            supported_steps (list[str]):
            machine (WorkerMachine | Unset): Informations non secrètes déclarées par le Worker ; le système est celui du
                conteneur.
            models (WorkerModels | Unset): Modèles utilisés par un worker, par rôle. Tous les rôles sont facultatifs. Jamais
                de secret, de jeton ni de clé d’API : les valeurs sont bornées, limitées à des caractères sûrs, et les valeurs
                ressemblant à un secret sont refusées.
     """

    capacity: int
    supported_steps: list[str]
    machine: WorkerMachine | Unset = UNSET
    models: WorkerModels | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_machine import WorkerMachine # noqa: PLC0415
        from ..models.worker_models import WorkerModels # noqa: PLC0415
        capacity = self.capacity

        supported_steps = self.supported_steps



        machine: dict[str, Any] | Unset = UNSET
        if not isinstance(self.machine, Unset):
            machine = self.machine.to_dict()

        models: dict[str, Any] | Unset = UNSET
        if not isinstance(self.models, Unset):
            models = self.models.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "capacity": capacity,
            "supported_steps": supported_steps,
        })
        if machine is not UNSET:
            field_dict["machine"] = machine
        if models is not UNSET:
            field_dict["models"] = models

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_machine import WorkerMachine # noqa: PLC0415
        from ..models.worker_models import WorkerModels # noqa: PLC0415
        d = dict(src_dict)
        capacity = d.pop("capacity")

        supported_steps = cast(list[str], d.pop("supported_steps"))


        _machine = d.pop("machine", UNSET)
        machine: WorkerMachine | Unset
        if isinstance(_machine,  Unset):
            machine = UNSET
        else:
            machine = WorkerMachine.from_dict(_machine)




        _models = d.pop("models", UNSET)
        models: WorkerModels | Unset
        if isinstance(_models,  Unset):
            models = UNSET
        else:
            models = WorkerModels.from_dict(_models)




        worker_pulse_input = cls(
            capacity=capacity,
            supported_steps=supported_steps,
            machine=machine,
            models=models,
        )

        return worker_pulse_input
