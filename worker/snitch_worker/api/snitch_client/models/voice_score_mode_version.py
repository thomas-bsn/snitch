from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_score_mode_version_mode import VoiceScoreModeVersionMode
from typing import cast
import datetime






T = TypeVar("T", bound="VoiceScoreModeVersion")



@_attrs_define
class VoiceScoreModeVersion:
    """ Une version du mode de score. La version 1 est le mode par défaut de Snitch.

        Attributes:
            version (int):
            mode (VoiceScoreModeVersionMode): cumulative : cosinus avec la Référence de la Personne, moyenne de ses
                échantillons contributifs validés pondérée par leur Parole nette plafonnée. best_sample : meilleur cosinus avec
                un seul de ses échantillons comparables.
            author (None | str): Administrateur qui a choisi ce mode ; null pour le mode par défaut.
            created_at (datetime.datetime):
     """

    version: int
    mode: VoiceScoreModeVersionMode
    author: None | str
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        version = self.version

        mode = self.mode.value

        author: None | str
        author = self.author

        created_at = self.created_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "version": version,
            "mode": mode,
            "author": author,
            "created_at": created_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version")

        mode = VoiceScoreModeVersionMode(d.pop("mode"))




        def _parse_author(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        author = _parse_author(d.pop("author"))


        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        voice_score_mode_version = cls(
            version=version,
            mode=mode,
            author=author,
            created_at=created_at,
        )


        voice_score_mode_version.additional_properties = d
        return voice_score_mode_version

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
