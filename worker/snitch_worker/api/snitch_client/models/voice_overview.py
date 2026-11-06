from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_overview_provider import VoiceOverviewProvider
from typing import cast

if TYPE_CHECKING:
  from ..models.voice_overview_active_models import VoiceOverviewActiveModels
  from ..models.voice_overview_worker import VoiceOverviewWorker





T = TypeVar("T", bound="VoiceOverview")



@_attrs_define
class VoiceOverview:
    """
        Attributes:
            provider (VoiceOverviewProvider):
            active_models (VoiceOverviewActiveModels):
            workers_total (int):
            workers_voice_ready (int):
            pending_voice_jobs (int): Travaux vocaux en attente tant qu'aucun Worker n'est prêt ; 0 sinon.
            workers (list[VoiceOverviewWorker]):
     """

    provider: VoiceOverviewProvider
    active_models: VoiceOverviewActiveModels
    workers_total: int
    workers_voice_ready: int
    pending_voice_jobs: int
    workers: list[VoiceOverviewWorker]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_overview_active_models import VoiceOverviewActiveModels # noqa: PLC0415
        from ..models.voice_overview_worker import VoiceOverviewWorker # noqa: PLC0415
        provider = self.provider.value

        active_models = self.active_models.to_dict()

        workers_total = self.workers_total

        workers_voice_ready = self.workers_voice_ready

        pending_voice_jobs = self.pending_voice_jobs

        workers = []
        for workers_item_data in self.workers:
            workers_item = workers_item_data.to_dict()
            workers.append(workers_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "provider": provider,
            "active_models": active_models,
            "workers_total": workers_total,
            "workers_voice_ready": workers_voice_ready,
            "pending_voice_jobs": pending_voice_jobs,
            "workers": workers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_overview_active_models import VoiceOverviewActiveModels # noqa: PLC0415
        from ..models.voice_overview_worker import VoiceOverviewWorker # noqa: PLC0415
        d = dict(src_dict)
        provider = VoiceOverviewProvider(d.pop("provider"))




        active_models = VoiceOverviewActiveModels.from_dict(d.pop("active_models"))




        workers_total = d.pop("workers_total")

        workers_voice_ready = d.pop("workers_voice_ready")

        pending_voice_jobs = d.pop("pending_voice_jobs")

        workers = []
        _workers = d.pop("workers")
        for workers_item_data in (_workers):
            workers_item = VoiceOverviewWorker.from_dict(workers_item_data)



            workers.append(workers_item)


        voice_overview = cls(
            provider=provider,
            active_models=active_models,
            workers_total=workers_total,
            workers_voice_ready=workers_voice_ready,
            pending_voice_jobs=pending_voice_jobs,
            workers=workers,
        )


        voice_overview.additional_properties = d
        return voice_overview

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
