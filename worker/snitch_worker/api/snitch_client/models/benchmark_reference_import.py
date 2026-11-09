from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="BenchmarkReferenceImport")



@_attrs_define
class BenchmarkReferenceImport:
    """
        Attributes:
            clip_id (int):
            stm (str | Unset): Texte de référence : `<fichier> <canal> <locuteur> <début> <fin> [<étiquettes>] texte`,
                lignes `;;` ignorées.
            rttm (str | Unset): Tours de parole de référence : `SPEAKER <fichier> <canal> <début> <durée> <NA> <NA>
                <locuteur> <NA> <NA>`.
     """

    clip_id: int
    stm: str | Unset = UNSET
    rttm: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        clip_id = self.clip_id

        stm = self.stm

        rttm = self.rttm


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "clip_id": clip_id,
        })
        if stm is not UNSET:
            field_dict["stm"] = stm
        if rttm is not UNSET:
            field_dict["rttm"] = rttm

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clip_id = d.pop("clip_id")

        stm = d.pop("stm", UNSET)

        rttm = d.pop("rttm", UNSET)

        benchmark_reference_import = cls(
            clip_id=clip_id,
            stm=stm,
            rttm=rttm,
        )


        benchmark_reference_import.additional_properties = d
        return benchmark_reference_import

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
