from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.runtime_binding_history_entry import RuntimeBindingHistoryEntry
  from ..models.runtime_binding_pending import RuntimeBindingPending





T = TypeVar("T", bound="RuntimeBinding")



@_attrs_define
class RuntimeBinding:
    """
        Attributes:
            worker_id (str):
            model_id (str):
            model_label (str):
            engine (str):
            family (str): Famille d'engine à la demande du modèle ; vide pour un modèle sur Release signée ou embarquée.
            versions (list[str]): Versions de la famille proposées pour une mise à jour (connues de Snitch, non révoquées,
                et la version par défaut) ; vide hors famille.
            kind (str):
            digest (str): Version liée au modèle ; vide si Snitch ne la connaît pas (modèle installé avant le suivi).
            version (str):
            revoked (bool): La liaison continue de servir avec un avertissement.
            previous_digest (str): Version précédente, cible du retour arrière ; vide si aucune.
            previous_version (str):
            previous_revoked (bool): Aucun retour arrière vers une version révoquée.
            history (list[RuntimeBindingHistoryEntry]):
            pending (RuntimeBindingPending | Unset):
     """

    worker_id: str
    model_id: str
    model_label: str
    engine: str
    family: str
    versions: list[str]
    kind: str
    digest: str
    version: str
    revoked: bool
    previous_digest: str
    previous_version: str
    previous_revoked: bool
    history: list[RuntimeBindingHistoryEntry]
    pending: RuntimeBindingPending | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.runtime_binding_history_entry import RuntimeBindingHistoryEntry # noqa: PLC0415
        from ..models.runtime_binding_pending import RuntimeBindingPending # noqa: PLC0415
        worker_id = self.worker_id

        model_id = self.model_id

        model_label = self.model_label

        engine = self.engine

        family = self.family

        versions = self.versions



        kind = self.kind

        digest = self.digest

        version = self.version

        revoked = self.revoked

        previous_digest = self.previous_digest

        previous_version = self.previous_version

        previous_revoked = self.previous_revoked

        history = []
        for history_item_data in self.history:
            history_item = history_item_data.to_dict()
            history.append(history_item)



        pending: dict[str, Any] | Unset = UNSET
        if not isinstance(self.pending, Unset):
            pending = self.pending.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "worker_id": worker_id,
            "model_id": model_id,
            "model_label": model_label,
            "engine": engine,
            "family": family,
            "versions": versions,
            "kind": kind,
            "digest": digest,
            "version": version,
            "revoked": revoked,
            "previous_digest": previous_digest,
            "previous_version": previous_version,
            "previous_revoked": previous_revoked,
            "history": history,
        })
        if pending is not UNSET:
            field_dict["pending"] = pending

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.runtime_binding_history_entry import RuntimeBindingHistoryEntry # noqa: PLC0415
        from ..models.runtime_binding_pending import RuntimeBindingPending # noqa: PLC0415
        d = dict(src_dict)
        worker_id = d.pop("worker_id")

        model_id = d.pop("model_id")

        model_label = d.pop("model_label")

        engine = d.pop("engine")

        family = d.pop("family")

        versions = cast(list[str], d.pop("versions"))


        kind = d.pop("kind")

        digest = d.pop("digest")

        version = d.pop("version")

        revoked = d.pop("revoked")

        previous_digest = d.pop("previous_digest")

        previous_version = d.pop("previous_version")

        previous_revoked = d.pop("previous_revoked")

        history = []
        _history = d.pop("history")
        for history_item_data in (_history):
            history_item = RuntimeBindingHistoryEntry.from_dict(history_item_data)



            history.append(history_item)


        _pending = d.pop("pending", UNSET)
        pending: RuntimeBindingPending | Unset
        if isinstance(_pending,  Unset):
            pending = UNSET
        else:
            pending = RuntimeBindingPending.from_dict(_pending)




        runtime_binding = cls(
            worker_id=worker_id,
            model_id=model_id,
            model_label=model_label,
            engine=engine,
            family=family,
            versions=versions,
            kind=kind,
            digest=digest,
            version=version,
            revoked=revoked,
            previous_digest=previous_digest,
            previous_version=previous_version,
            previous_revoked=previous_revoked,
            history=history,
            pending=pending,
        )


        runtime_binding.additional_properties = d
        return runtime_binding

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
