from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_trial_config import VoiceTrialConfig
  from ..models.worker_voice_trial_input_trial import WorkerVoiceTrialInputTrial





T = TypeVar("T", bound="WorkerVoiceTrialInput")



@_attrs_define
class WorkerVoiceTrialInput:
    """ Entrée d'un essai de diarisation (étapes voicetrial.*) : le modèle du catalogue et les options choisis pour l'essai,
    jamais la configuration active ni une version de pipeline.

        Attributes:
            config (VoiceTrialConfig):
            trial (WorkerVoiceTrialInputTrial):
     """

    config: VoiceTrialConfig
    trial: WorkerVoiceTrialInputTrial
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_trial_config import VoiceTrialConfig # noqa: PLC0415
        from ..models.worker_voice_trial_input_trial import WorkerVoiceTrialInputTrial # noqa: PLC0415
        config = self.config.to_dict()

        trial = self.trial.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "config": config,
            "trial": trial,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_trial_config import VoiceTrialConfig # noqa: PLC0415
        from ..models.worker_voice_trial_input_trial import WorkerVoiceTrialInputTrial # noqa: PLC0415
        d = dict(src_dict)
        config = VoiceTrialConfig.from_dict(d.pop("config"))




        trial = WorkerVoiceTrialInputTrial.from_dict(d.pop("trial"))




        worker_voice_trial_input = cls(
            config=config,
            trial=trial,
        )


        worker_voice_trial_input.additional_properties = d
        return worker_voice_trial_input

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
