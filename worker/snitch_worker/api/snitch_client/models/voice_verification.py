from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_verification_case import VoiceVerificationCase
  from ..models.voice_verification_counts import VoiceVerificationCounts





T = TypeVar("T", bound="VoiceVerification")



@_attrs_define
class VoiceVerification:
    """
        Attributes:
            items (list[VoiceVerificationCase]):
            total (int): Nombre de cas avant pagination.
            counts (VoiceVerificationCounts): Locuteurs locaux des clips visibles par niveau de décision effectif, calculés
                en base.
     """

    items: list[VoiceVerificationCase]
    total: int
    counts: VoiceVerificationCounts
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_verification_case import VoiceVerificationCase # noqa: PLC0415
        from ..models.voice_verification_counts import VoiceVerificationCounts # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        total = self.total

        counts = self.counts.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "items": items,
            "total": total,
            "counts": counts,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_verification_case import VoiceVerificationCase # noqa: PLC0415
        from ..models.voice_verification_counts import VoiceVerificationCounts # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = VoiceVerificationCase.from_dict(items_item_data)



            items.append(items_item)


        total = d.pop("total")

        counts = VoiceVerificationCounts.from_dict(d.pop("counts"))




        voice_verification = cls(
            items=items,
            total=total,
            counts=counts,
        )


        voice_verification.additional_properties = d
        return voice_verification

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
