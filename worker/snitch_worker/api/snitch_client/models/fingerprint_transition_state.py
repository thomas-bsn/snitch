from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.fingerprint_transition import FingerprintTransition





T = TypeVar("T", bound="FingerprintTransitionState")



@_attrs_define
class FingerprintTransitionState:
    """
        Attributes:
            open_ (bool):
            transition (FingerprintTransition | Unset):
     """

    open_: bool
    transition: FingerprintTransition | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.fingerprint_transition import FingerprintTransition # noqa: PLC0415
        open_ = self.open_

        transition: dict[str, Any] | Unset = UNSET
        if not isinstance(self.transition, Unset):
            transition = self.transition.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "open": open_,
        })
        if transition is not UNSET:
            field_dict["transition"] = transition

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.fingerprint_transition import FingerprintTransition # noqa: PLC0415
        d = dict(src_dict)
        open_ = d.pop("open")

        _transition = d.pop("transition", UNSET)
        transition: FingerprintTransition | Unset
        if isinstance(_transition,  Unset):
            transition = UNSET
        else:
            transition = FingerprintTransition.from_dict(_transition)




        fingerprint_transition_state = cls(
            open_=open_,
            transition=transition,
        )

        return fingerprint_transition_state
