from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.worker_model import WorkerModel





T = TypeVar("T", bound="WorkerModels")



@_attrs_define
class WorkerModels:
    """ Modèles utilisés par un worker, par rôle. Tous les rôles sont facultatifs. Jamais de secret, de jeton ni de clé
    d’API : les valeurs sont bornées, limitées à des caractères sûrs, et les valeurs ressemblant à un secret sont
    refusées.

        Attributes:
            transcription (WorkerModel | Unset):
            diarization (WorkerModel | Unset):
            fingerprint (WorkerModel | Unset):
     """

    transcription: WorkerModel | Unset = UNSET
    diarization: WorkerModel | Unset = UNSET
    fingerprint: WorkerModel | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_model import WorkerModel # noqa: PLC0415
        transcription: dict[str, Any] | Unset = UNSET
        if not isinstance(self.transcription, Unset):
            transcription = self.transcription.to_dict()

        diarization: dict[str, Any] | Unset = UNSET
        if not isinstance(self.diarization, Unset):
            diarization = self.diarization.to_dict()

        fingerprint: dict[str, Any] | Unset = UNSET
        if not isinstance(self.fingerprint, Unset):
            fingerprint = self.fingerprint.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if transcription is not UNSET:
            field_dict["transcription"] = transcription
        if diarization is not UNSET:
            field_dict["diarization"] = diarization
        if fingerprint is not UNSET:
            field_dict["fingerprint"] = fingerprint

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_model import WorkerModel # noqa: PLC0415
        d = dict(src_dict)
        _transcription = d.pop("transcription", UNSET)
        transcription: WorkerModel | Unset
        if isinstance(_transcription,  Unset):
            transcription = UNSET
        else:
            transcription = WorkerModel.from_dict(_transcription)




        _diarization = d.pop("diarization", UNSET)
        diarization: WorkerModel | Unset
        if isinstance(_diarization,  Unset):
            diarization = UNSET
        else:
            diarization = WorkerModel.from_dict(_diarization)




        _fingerprint = d.pop("fingerprint", UNSET)
        fingerprint: WorkerModel | Unset
        if isinstance(_fingerprint,  Unset):
            fingerprint = UNSET
        else:
            fingerprint = WorkerModel.from_dict(_fingerprint)




        worker_models = cls(
            transcription=transcription,
            diarization=diarization,
            fingerprint=fingerprint,
        )

        return worker_models
