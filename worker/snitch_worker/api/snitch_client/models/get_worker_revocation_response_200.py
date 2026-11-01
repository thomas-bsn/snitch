from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="GetWorkerRevocationResponse200")



@_attrs_define
class GetWorkerRevocationResponse200:
    """
        Attributes:
            revoked (bool):
            revoked_at (datetime.datetime | Unset):
     """

    revoked: bool
    revoked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        revoked = self.revoked

        revoked_at: str | Unset = UNSET
        if not isinstance(self.revoked_at, Unset):
            revoked_at = self.revoked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "revoked": revoked,
        })
        if revoked_at is not UNSET:
            field_dict["revoked_at"] = revoked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        revoked = d.pop("revoked")

        _revoked_at = d.pop("revoked_at", UNSET)
        revoked_at: datetime.datetime | Unset
        if isinstance(_revoked_at,  Unset):
            revoked_at = UNSET
        else:
            revoked_at = datetime.datetime.fromisoformat(_revoked_at)




        get_worker_revocation_response_200 = cls(
            revoked=revoked,
            revoked_at=revoked_at,
        )

        return get_worker_revocation_response_200
