from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.clip_analysis_run import ClipAnalysisRun
  from ..models.clip_reanalysis_stake import ClipReanalysisStake





T = TypeVar("T", bound="ClipAnalysisConflict")



@_attrs_define
class ClipAnalysisConflict:
    """
        Attributes:
            error (str):
            run (ClipAnalysisRun | Unset):
            stake (ClipReanalysisStake | Unset): Ce que la réanalyse rendrait obsolète ; rien n'est supprimé (historiques,
                échantillons validés et anciens résultats conservés).
            stake_at (datetime.datetime | Unset): Instant où l'enjeu a été lu ; à renvoyer en confirmed_stake_at.
     """

    error: str
    run: ClipAnalysisRun | Unset = UNSET
    stake: ClipReanalysisStake | Unset = UNSET
    stake_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.clip_analysis_run import ClipAnalysisRun # noqa: PLC0415
        from ..models.clip_reanalysis_stake import ClipReanalysisStake # noqa: PLC0415
        error = self.error

        run: dict[str, Any] | Unset = UNSET
        if not isinstance(self.run, Unset):
            run = self.run.to_dict()

        stake: dict[str, Any] | Unset = UNSET
        if not isinstance(self.stake, Unset):
            stake = self.stake.to_dict()

        stake_at: str | Unset = UNSET
        if not isinstance(self.stake_at, Unset):
            stake_at = self.stake_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "error": error,
        })
        if run is not UNSET:
            field_dict["run"] = run
        if stake is not UNSET:
            field_dict["stake"] = stake
        if stake_at is not UNSET:
            field_dict["stake_at"] = stake_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.clip_analysis_run import ClipAnalysisRun # noqa: PLC0415
        from ..models.clip_reanalysis_stake import ClipReanalysisStake # noqa: PLC0415
        d = dict(src_dict)
        error = d.pop("error")

        _run = d.pop("run", UNSET)
        run: ClipAnalysisRun | Unset
        if isinstance(_run,  Unset):
            run = UNSET
        else:
            run = ClipAnalysisRun.from_dict(_run)




        _stake = d.pop("stake", UNSET)
        stake: ClipReanalysisStake | Unset
        if isinstance(_stake,  Unset):
            stake = UNSET
        else:
            stake = ClipReanalysisStake.from_dict(_stake)




        _stake_at = d.pop("stake_at", UNSET)
        stake_at: datetime.datetime | Unset
        if isinstance(_stake_at,  Unset):
            stake_at = UNSET
        else:
            stake_at = datetime.datetime.fromisoformat(_stake_at)




        clip_analysis_conflict = cls(
            error=error,
            run=run,
            stake=stake,
            stake_at=stake_at,
        )


        clip_analysis_conflict.additional_properties = d
        return clip_analysis_conflict

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
