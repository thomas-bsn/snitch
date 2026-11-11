from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_eligibility_group_preview import VoiceEligibilityGroupPreview





T = TypeVar("T", bound="VoiceEligibilityPreview")



@_attrs_define
class VoiceEligibilityPreview:
    """ Aperçu sans effet de l'éligibilité des empreintes enregistrées (#394) : mêmes calculs que la requalification.

        Attributes:
            active_version (int): Version de la règle active.
            persons_with_samples (int): Personnes ayant au moins un Échantillon vocal validé actif.
            groups (list[VoiceEligibilityGroupPreview]):
     """

    active_version: int
    persons_with_samples: int
    groups: list[VoiceEligibilityGroupPreview]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_eligibility_group_preview import VoiceEligibilityGroupPreview # noqa: PLC0415
        active_version = self.active_version

        persons_with_samples = self.persons_with_samples

        groups = []
        for groups_item_data in self.groups:
            groups_item = groups_item_data.to_dict()
            groups.append(groups_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "active_version": active_version,
            "persons_with_samples": persons_with_samples,
            "groups": groups,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_eligibility_group_preview import VoiceEligibilityGroupPreview # noqa: PLC0415
        d = dict(src_dict)
        active_version = d.pop("active_version")

        persons_with_samples = d.pop("persons_with_samples")

        groups = []
        _groups = d.pop("groups")
        for groups_item_data in (_groups):
            groups_item = VoiceEligibilityGroupPreview.from_dict(groups_item_data)



            groups.append(groups_item)


        voice_eligibility_preview = cls(
            active_version=active_version,
            persons_with_samples=persons_with_samples,
            groups=groups,
        )


        voice_eligibility_preview.additional_properties = d
        return voice_eligibility_preview

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
