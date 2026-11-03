from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.post_worker_update_status_body_state import PostWorkerUpdateStatusBodyState
from ..types import UNSET, Unset






T = TypeVar("T", bound="PostWorkerUpdateStatusBody")



@_attrs_define
class PostWorkerUpdateStatusBody:
    """
        Attributes:
            state (PostWorkerUpdateStatusBodyState):
            version (str):
            digest (str | Unset):
            cause_code (str | Unset):
     """

    state: PostWorkerUpdateStatusBodyState
    version: str
    digest: str | Unset = UNSET
    cause_code: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        state = self.state.value

        version = self.version

        digest = self.digest

        cause_code = self.cause_code


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "state": state,
            "version": version,
        })
        if digest is not UNSET:
            field_dict["digest"] = digest
        if cause_code is not UNSET:
            field_dict["cause_code"] = cause_code

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        state = PostWorkerUpdateStatusBodyState(d.pop("state"))




        version = d.pop("version")

        digest = d.pop("digest", UNSET)

        cause_code = d.pop("cause_code", UNSET)

        post_worker_update_status_body = cls(
            state=state,
            version=version,
            digest=digest,
            cause_code=cause_code,
        )

        return post_worker_update_status_body
