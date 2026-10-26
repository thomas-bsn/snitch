from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.corrections_to_replace import CorrectionsToReplace





T = TypeVar("T", bound="CorrectionsWarning")



@_attrs_define
class CorrectionsWarning:
    """
        Attributes:
            error (str):
            corrections (CorrectionsToReplace | Unset): Corrections humaines actives de la transcription visible d'un clip,
                que la retranscription remplacerait (elles resteraient dans l'historique, obsolètes).
     """

    error: str
    corrections: CorrectionsToReplace | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.corrections_to_replace import CorrectionsToReplace # noqa: PLC0415
        error = self.error

        corrections: dict[str, Any] | Unset = UNSET
        if not isinstance(self.corrections, Unset):
            corrections = self.corrections.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "error": error,
        })
        if corrections is not UNSET:
            field_dict["corrections"] = corrections

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corrections_to_replace import CorrectionsToReplace # noqa: PLC0415
        d = dict(src_dict)
        error = d.pop("error")

        _corrections = d.pop("corrections", UNSET)
        corrections: CorrectionsToReplace | Unset
        if isinstance(_corrections,  Unset):
            corrections = UNSET
        else:
            corrections = CorrectionsToReplace.from_dict(_corrections)




        corrections_warning = cls(
            error=error,
            corrections=corrections,
        )


        corrections_warning.additional_properties = d
        return corrections_warning

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
