from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="VoiceProfileDeletion")



@_attrs_define
class VoiceProfileDeletion:
    """
        Attributes:
            samples (int): Échantillons vocaux validés effacés (actifs et retirés).
            fingerprints (int): Empreintes effacées.
            attributions (int): Attributions effacées ; les voix correspondantes sont inconnues.
            deleted_at (datetime.datetime):
     """

    samples: int
    fingerprints: int
    attributions: int
    deleted_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        samples = self.samples

        fingerprints = self.fingerprints

        attributions = self.attributions

        deleted_at = self.deleted_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "samples": samples,
            "fingerprints": fingerprints,
            "attributions": attributions,
            "deleted_at": deleted_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        samples = d.pop("samples")

        fingerprints = d.pop("fingerprints")

        attributions = d.pop("attributions")

        deleted_at = datetime.datetime.fromisoformat(d.pop("deleted_at"))




        voice_profile_deletion = cls(
            samples=samples,
            fingerprints=fingerprints,
            attributions=attributions,
            deleted_at=deleted_at,
        )


        voice_profile_deletion.additional_properties = d
        return voice_profile_deletion

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
