from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_calibration_mode_mode import VoiceCalibrationModeMode
from ..models.voice_calibration_mode_withheld import VoiceCalibrationModeWithheld
from typing import cast

if TYPE_CHECKING:
  from ..models.voice_score_bin import VoiceScoreBin
  from ..models.voice_threshold_proposal import VoiceThresholdProposal
  from ..models.voice_threshold_sweep import VoiceThresholdSweep





T = TypeVar("T", bound="VoiceCalibrationMode")



@_attrs_define
class VoiceCalibrationMode:
    """ Mesure d'un mode de score sur le groupe actif.

        Attributes:
            mode (VoiceCalibrationModeMode):
            tests (int): Tests ayant une empreinte comparable dans le groupe ; chacun est aussi rejoué comme inconnu simulé.
            persons (int): Personnes ayant une Référence dans ce mode et au moins un test.
            same_person_scores (int):
            other_person_scores (int):
            distribution (list[VoiceScoreBin]):
            sweep (list[VoiceThresholdSweep]): Seuil automatique de 0,05 à 1 par pas de 0,05.
            current (None | VoiceThresholdSweep): Chiffres aux seuils appliqués au groupe ; null sans seuils appliqués.
            proposal (None | VoiceThresholdProposal):
            withheld (VoiceCalibrationModeWithheld): Raison de l'absence de proposition : too_few (effectifs trop faibles),
                no_safe_threshold (aucun seuil ne garde la précision sans reconnaître d'inconnu).
     """

    mode: VoiceCalibrationModeMode
    tests: int
    persons: int
    same_person_scores: int
    other_person_scores: int
    distribution: list[VoiceScoreBin]
    sweep: list[VoiceThresholdSweep]
    current: None | VoiceThresholdSweep
    proposal: None | VoiceThresholdProposal
    withheld: VoiceCalibrationModeWithheld
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_score_bin import VoiceScoreBin # noqa: PLC0415
        from ..models.voice_threshold_proposal import VoiceThresholdProposal # noqa: PLC0415
        from ..models.voice_threshold_sweep import VoiceThresholdSweep # noqa: PLC0415
        mode = self.mode.value

        tests = self.tests

        persons = self.persons

        same_person_scores = self.same_person_scores

        other_person_scores = self.other_person_scores

        distribution = []
        for distribution_item_data in self.distribution:
            distribution_item = distribution_item_data.to_dict()
            distribution.append(distribution_item)



        sweep = []
        for sweep_item_data in self.sweep:
            sweep_item = sweep_item_data.to_dict()
            sweep.append(sweep_item)



        current: dict[str, Any] | None
        if isinstance(self.current, VoiceThresholdSweep):
            current = self.current.to_dict()
        else:
            current = self.current

        proposal: dict[str, Any] | None
        if isinstance(self.proposal, VoiceThresholdProposal):
            proposal = self.proposal.to_dict()
        else:
            proposal = self.proposal

        withheld = self.withheld.value


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "mode": mode,
            "tests": tests,
            "persons": persons,
            "same_person_scores": same_person_scores,
            "other_person_scores": other_person_scores,
            "distribution": distribution,
            "sweep": sweep,
            "current": current,
            "proposal": proposal,
            "withheld": withheld,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_score_bin import VoiceScoreBin # noqa: PLC0415
        from ..models.voice_threshold_proposal import VoiceThresholdProposal # noqa: PLC0415
        from ..models.voice_threshold_sweep import VoiceThresholdSweep # noqa: PLC0415
        d = dict(src_dict)
        mode = VoiceCalibrationModeMode(d.pop("mode"))




        tests = d.pop("tests")

        persons = d.pop("persons")

        same_person_scores = d.pop("same_person_scores")

        other_person_scores = d.pop("other_person_scores")

        distribution = []
        _distribution = d.pop("distribution")
        for distribution_item_data in (_distribution):
            distribution_item = VoiceScoreBin.from_dict(distribution_item_data)



            distribution.append(distribution_item)


        sweep = []
        _sweep = d.pop("sweep")
        for sweep_item_data in (_sweep):
            sweep_item = VoiceThresholdSweep.from_dict(sweep_item_data)



            sweep.append(sweep_item)


        def _parse_current(data: object) -> None | VoiceThresholdSweep:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                current_type_1 = VoiceThresholdSweep.from_dict(data)



                return current_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | VoiceThresholdSweep, data)

        current = _parse_current(d.pop("current"))


        def _parse_proposal(data: object) -> None | VoiceThresholdProposal:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                proposal_type_1 = VoiceThresholdProposal.from_dict(data)



                return proposal_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | VoiceThresholdProposal, data)

        proposal = _parse_proposal(d.pop("proposal"))


        withheld = VoiceCalibrationModeWithheld(d.pop("withheld"))




        voice_calibration_mode = cls(
            mode=mode,
            tests=tests,
            persons=persons,
            same_person_scores=same_person_scores,
            other_person_scores=other_person_scores,
            distribution=distribution,
            sweep=sweep,
            current=current,
            proposal=proposal,
            withheld=withheld,
        )


        voice_calibration_mode.additional_properties = d
        return voice_calibration_mode

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
