from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_threshold_sweep import VoiceThresholdSweep





T = TypeVar("T", bound="VoiceThresholdProposal")



@_attrs_define
class VoiceThresholdProposal:
    """ Proposition de seuils : jamais appliquée sans action de l'administrateur.

        Attributes:
            suggestion_threshold (float):
            automatic_threshold (float): Plus bas seuil, au centième, à partir duquel la mesure garde la précision minimale
                des critères d'activation et ne reconnaît aucun inconnu simulé.
            ambiguity_margin (float):
            figures (VoiceThresholdSweep): Décision rejouée sur les tests à un jeu de seuils.
     """

    suggestion_threshold: float
    automatic_threshold: float
    ambiguity_margin: float
    figures: VoiceThresholdSweep
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_threshold_sweep import VoiceThresholdSweep # noqa: PLC0415
        suggestion_threshold = self.suggestion_threshold

        automatic_threshold = self.automatic_threshold

        ambiguity_margin = self.ambiguity_margin

        figures = self.figures.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "suggestion_threshold": suggestion_threshold,
            "automatic_threshold": automatic_threshold,
            "ambiguity_margin": ambiguity_margin,
            "figures": figures,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_threshold_sweep import VoiceThresholdSweep # noqa: PLC0415
        d = dict(src_dict)
        suggestion_threshold = d.pop("suggestion_threshold")

        automatic_threshold = d.pop("automatic_threshold")

        ambiguity_margin = d.pop("ambiguity_margin")

        figures = VoiceThresholdSweep.from_dict(d.pop("figures"))




        voice_threshold_proposal = cls(
            suggestion_threshold=suggestion_threshold,
            automatic_threshold=automatic_threshold,
            ambiguity_margin=ambiguity_margin,
            figures=figures,
        )


        voice_threshold_proposal.additional_properties = d
        return voice_threshold_proposal

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
