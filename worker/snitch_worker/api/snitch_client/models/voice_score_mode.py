from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_score_mode_mode import VoiceScoreModeMode
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.voice_score_mode_version import VoiceScoreModeVersion





T = TypeVar("T", bound="VoiceScoreMode")



@_attrs_define
class VoiceScoreMode:
    """ Mode de score du rapprochement (ADR 0064) : version active et historique.

        Attributes:
            version (int):
            mode (VoiceScoreModeMode): cumulative : cosinus avec la Référence de la Personne, moyenne de ses échantillons
                contributifs validés pondérée par leur Parole nette plafonnée. best_sample : meilleur cosinus avec un seul de
                ses échantillons comparables.
            author (None | str): Administrateur qui a choisi ce mode ; null pour le mode par défaut.
            created_at (datetime.datetime):
            history (list[VoiceScoreModeVersion]): Vingt dernières versions, la plus récente d'abord.
     """

    version: int
    mode: VoiceScoreModeMode
    author: None | str
    created_at: datetime.datetime
    history: list[VoiceScoreModeVersion]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_score_mode_version import VoiceScoreModeVersion # noqa: PLC0415
        version = self.version

        mode = self.mode.value

        author: None | str
        author = self.author

        created_at = self.created_at.isoformat()

        history = []
        for history_item_data in self.history:
            history_item = history_item_data.to_dict()
            history.append(history_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "version": version,
            "mode": mode,
            "author": author,
            "created_at": created_at,
            "history": history,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_score_mode_version import VoiceScoreModeVersion # noqa: PLC0415
        d = dict(src_dict)
        version = d.pop("version")

        mode = VoiceScoreModeMode(d.pop("mode"))




        def _parse_author(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        author = _parse_author(d.pop("author"))


        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        history = []
        _history = d.pop("history")
        for history_item_data in (_history):
            history_item = VoiceScoreModeVersion.from_dict(history_item_data)



            history.append(history_item)


        voice_score_mode = cls(
            version=version,
            mode=mode,
            author=author,
            created_at=created_at,
            history=history,
        )


        voice_score_mode.additional_properties = d
        return voice_score_mode

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
