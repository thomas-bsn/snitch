from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.clip_analysis_deletion_stake import ClipAnalysisDeletionStake





T = TypeVar("T", bound="GetClipAnalysisDeletionPreviewResponse200")



@_attrs_define
class GetClipAnalysisDeletionPreviewResponse200:
    """
        Attributes:
            steps (list[str]):
            stake (ClipAnalysisDeletionStake): Pertes d'une suppression d'analyse ; rien n'est supprimé par ce calcul. Les
                Personnes et leurs échantillons validés ne sont jamais supprimés, seulement retirés de la référence.
     """

    steps: list[str]
    stake: ClipAnalysisDeletionStake
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.clip_analysis_deletion_stake import ClipAnalysisDeletionStake # noqa: PLC0415
        steps = self.steps



        stake = self.stake.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "steps": steps,
            "stake": stake,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.clip_analysis_deletion_stake import ClipAnalysisDeletionStake # noqa: PLC0415
        d = dict(src_dict)
        steps = cast(list[str], d.pop("steps"))


        stake = ClipAnalysisDeletionStake.from_dict(d.pop("stake"))




        get_clip_analysis_deletion_preview_response_200 = cls(
            steps=steps,
            stake=stake,
        )


        get_clip_analysis_deletion_preview_response_200.additional_properties = d
        return get_clip_analysis_deletion_preview_response_200

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
