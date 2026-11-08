from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_engine_manifest import VoiceEngineManifest





T = TypeVar("T", bound="EngineCleanup")



@_attrs_define
class EngineCleanup:
    """
        Attributes:
            order_id (str):
            digest (str):
            runtime (VoiceEngineManifest):
     """

    order_id: str
    digest: str
    runtime: VoiceEngineManifest
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_engine_manifest import VoiceEngineManifest # noqa: PLC0415
        order_id = self.order_id

        digest = self.digest

        runtime = self.runtime.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "order_id": order_id,
            "digest": digest,
            "runtime": runtime,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_engine_manifest import VoiceEngineManifest # noqa: PLC0415
        d = dict(src_dict)
        order_id = d.pop("order_id")

        digest = d.pop("digest")

        runtime = VoiceEngineManifest.from_dict(d.pop("runtime"))




        engine_cleanup = cls(
            order_id=order_id,
            digest=digest,
            runtime=runtime,
        )


        engine_cleanup.additional_properties = d
        return engine_cleanup

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
