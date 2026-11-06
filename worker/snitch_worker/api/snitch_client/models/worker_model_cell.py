from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_model_cell_blocked import WorkerModelCellBlocked
from ..models.worker_model_cell_cause_code import WorkerModelCellCauseCode
from ..models.worker_model_cell_kind import WorkerModelCellKind
from ..models.worker_model_cell_state import WorkerModelCellState






T = TypeVar("T", bound="WorkerModelCell")



@_attrs_define
class WorkerModelCell:
    """
        Attributes:
            kind (WorkerModelCellKind):
            supported (bool):
            model_id (str):
            label (str):
            repo (str):
            revision (str):
            state (WorkerModelCellState): Pendant un remplacement la cellule montre le nouveau modèle : switching (prêt,
                bascule décidée, retrait de l'ancien en attente), removing (ordre remove de l'ancien envoyé).
            cause_code (WorkerModelCellCauseCode):
            bytes_received (int):
            bytes_total (int):
            occupied_bytes (int):
            order_id (str):
            blocked (WorkerModelCellBlocked): Ordre d'installation en attente depuis plus de deux minutes : Worker
                déconnecté (worker_offline) ou connecté mais qui ne l'a pas tiré (order_not_pulled). Vide sinon.
            replaces_model_id (str): Modèle que celui-ci remplace sur ce Worker ; vide hors remplacement. L'ancien reste
                actif et installé jusqu'à son retrait.
            waiting_for_jobs (bool): Le retrait de l'ancien modèle est différé parce qu'un travail en cours l'utilise.
     """

    kind: WorkerModelCellKind
    supported: bool
    model_id: str
    label: str
    repo: str
    revision: str
    state: WorkerModelCellState
    cause_code: WorkerModelCellCauseCode
    bytes_received: int
    bytes_total: int
    occupied_bytes: int
    order_id: str
    blocked: WorkerModelCellBlocked
    replaces_model_id: str
    waiting_for_jobs: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        supported = self.supported

        model_id = self.model_id

        label = self.label

        repo = self.repo

        revision = self.revision

        state = self.state.value

        cause_code = self.cause_code.value

        bytes_received = self.bytes_received

        bytes_total = self.bytes_total

        occupied_bytes = self.occupied_bytes

        order_id = self.order_id

        blocked = self.blocked.value

        replaces_model_id = self.replaces_model_id

        waiting_for_jobs = self.waiting_for_jobs


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "kind": kind,
            "supported": supported,
            "model_id": model_id,
            "label": label,
            "repo": repo,
            "revision": revision,
            "state": state,
            "cause_code": cause_code,
            "bytes_received": bytes_received,
            "bytes_total": bytes_total,
            "occupied_bytes": occupied_bytes,
            "order_id": order_id,
            "blocked": blocked,
            "replaces_model_id": replaces_model_id,
            "waiting_for_jobs": waiting_for_jobs,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = WorkerModelCellKind(d.pop("kind"))




        supported = d.pop("supported")

        model_id = d.pop("model_id")

        label = d.pop("label")

        repo = d.pop("repo")

        revision = d.pop("revision")

        state = WorkerModelCellState(d.pop("state"))




        cause_code = WorkerModelCellCauseCode(d.pop("cause_code"))




        bytes_received = d.pop("bytes_received")

        bytes_total = d.pop("bytes_total")

        occupied_bytes = d.pop("occupied_bytes")

        order_id = d.pop("order_id")

        blocked = WorkerModelCellBlocked(d.pop("blocked"))




        replaces_model_id = d.pop("replaces_model_id")

        waiting_for_jobs = d.pop("waiting_for_jobs")

        worker_model_cell = cls(
            kind=kind,
            supported=supported,
            model_id=model_id,
            label=label,
            repo=repo,
            revision=revision,
            state=state,
            cause_code=cause_code,
            bytes_received=bytes_received,
            bytes_total=bytes_total,
            occupied_bytes=occupied_bytes,
            order_id=order_id,
            blocked=blocked,
            replaces_model_id=replaces_model_id,
            waiting_for_jobs=waiting_for_jobs,
        )


        worker_model_cell.additional_properties = d
        return worker_model_cell

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
