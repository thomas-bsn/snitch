from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.benchmark_reference_import import BenchmarkReferenceImport





T = TypeVar("T", bound="BenchmarkDatasetInput")



@_attrs_define
class BenchmarkDatasetInput:
    """
        Attributes:
            name (str):
            test_clip_ids (list[int]):
            reference_clip_ids (list[int]):
            verified_clip_ids (list[int]): Clips de test dont l'administrateur atteste que le texte et les locuteurs ont été
                relus en entier.
            reference_imports (list[BenchmarkReferenceImport] | Unset): Références externes (STM et/ou RTTM, 2 Mo au total)
                qui remplacent la référence de production d'un clip de test ; le clip compte alors comme certifié. La piste est
                celle de la transcription de production du clip.
     """

    name: str
    test_clip_ids: list[int]
    reference_clip_ids: list[int]
    verified_clip_ids: list[int]
    reference_imports: list[BenchmarkReferenceImport] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.benchmark_reference_import import BenchmarkReferenceImport # noqa: PLC0415
        name = self.name

        test_clip_ids = self.test_clip_ids



        reference_clip_ids = self.reference_clip_ids



        verified_clip_ids = self.verified_clip_ids



        reference_imports: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.reference_imports, Unset):
            reference_imports = []
            for reference_imports_item_data in self.reference_imports:
                reference_imports_item = reference_imports_item_data.to_dict()
                reference_imports.append(reference_imports_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "name": name,
            "test_clip_ids": test_clip_ids,
            "reference_clip_ids": reference_clip_ids,
            "verified_clip_ids": verified_clip_ids,
        })
        if reference_imports is not UNSET:
            field_dict["reference_imports"] = reference_imports

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.benchmark_reference_import import BenchmarkReferenceImport # noqa: PLC0415
        d = dict(src_dict)
        name = d.pop("name")

        test_clip_ids = cast(list[int], d.pop("test_clip_ids"))


        reference_clip_ids = cast(list[int], d.pop("reference_clip_ids"))


        verified_clip_ids = cast(list[int], d.pop("verified_clip_ids"))


        _reference_imports = d.pop("reference_imports", UNSET)
        reference_imports: list[BenchmarkReferenceImport] | Unset = UNSET
        if _reference_imports is not UNSET:
            reference_imports = []
            for reference_imports_item_data in _reference_imports:
                reference_imports_item = BenchmarkReferenceImport.from_dict(reference_imports_item_data)



                reference_imports.append(reference_imports_item)


        benchmark_dataset_input = cls(
            name=name,
            test_clip_ids=test_clip_ids,
            reference_clip_ids=reference_clip_ids,
            verified_clip_ids=verified_clip_ids,
            reference_imports=reference_imports,
        )


        benchmark_dataset_input.additional_properties = d
        return benchmark_dataset_input

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
