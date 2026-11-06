from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_model_inspection_compatible_kinds_item import VoiceModelInspectionCompatibleKindsItem
from typing import cast






T = TypeVar("T", bound="VoiceModelInspection")



@_attrs_define
class VoiceModelInspection:
    """
        Attributes:
            repo (str):
            revision (str): Commit lu, épinglé à l'ajout.
            license_ (str):
            gated (bool):
            engine (str): Moteur déduit de la matrice ; vide si aucun adaptateur approuvé.
            proposed_kind (None | str): Null quand les métadonnées ne tranchent pas : la personne choisit parmi les trois
                types.
            compatible_kinds (list[VoiceModelInspectionCompatibleKindsItem]):
            supported (bool):
            unsupported_reason (str):
            existing_ids (list[str]): Références du catalogue déjà créées pour ce dépôt.
     """

    repo: str
    revision: str
    license_: str
    gated: bool
    engine: str
    proposed_kind: None | str
    compatible_kinds: list[VoiceModelInspectionCompatibleKindsItem]
    supported: bool
    unsupported_reason: str
    existing_ids: list[str]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        repo = self.repo

        revision = self.revision

        license_ = self.license_

        gated = self.gated

        engine = self.engine

        proposed_kind: None | str
        proposed_kind = self.proposed_kind

        compatible_kinds = []
        for compatible_kinds_item_data in self.compatible_kinds:
            compatible_kinds_item = compatible_kinds_item_data.value
            compatible_kinds.append(compatible_kinds_item)



        supported = self.supported

        unsupported_reason = self.unsupported_reason

        existing_ids = self.existing_ids




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "repo": repo,
            "revision": revision,
            "license": license_,
            "gated": gated,
            "engine": engine,
            "proposed_kind": proposed_kind,
            "compatible_kinds": compatible_kinds,
            "supported": supported,
            "unsupported_reason": unsupported_reason,
            "existing_ids": existing_ids,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        repo = d.pop("repo")

        revision = d.pop("revision")

        license_ = d.pop("license")

        gated = d.pop("gated")

        engine = d.pop("engine")

        def _parse_proposed_kind(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        proposed_kind = _parse_proposed_kind(d.pop("proposed_kind"))


        compatible_kinds = []
        _compatible_kinds = d.pop("compatible_kinds")
        for compatible_kinds_item_data in (_compatible_kinds):
            compatible_kinds_item = VoiceModelInspectionCompatibleKindsItem(compatible_kinds_item_data)



            compatible_kinds.append(compatible_kinds_item)


        supported = d.pop("supported")

        unsupported_reason = d.pop("unsupported_reason")

        existing_ids = cast(list[str], d.pop("existing_ids"))


        voice_model_inspection = cls(
            repo=repo,
            revision=revision,
            license_=license_,
            gated=gated,
            engine=engine,
            proposed_kind=proposed_kind,
            compatible_kinds=compatible_kinds,
            supported=supported,
            unsupported_reason=unsupported_reason,
            existing_ids=existing_ids,
        )


        voice_model_inspection.additional_properties = d
        return voice_model_inspection

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
