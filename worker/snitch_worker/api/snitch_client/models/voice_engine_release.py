from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_engine_manifest import VoiceEngineManifest





T = TypeVar("T", bound="VoiceEngineRelease")



@_attrs_define
class VoiceEngineRelease:
    """
        Attributes:
            digest (str):
            manifest (VoiceEngineManifest):
            revoked (bool):
     """

    digest: str
    manifest: VoiceEngineManifest
    revoked: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_engine_manifest import VoiceEngineManifest # noqa: PLC0415
        digest = self.digest

        manifest = self.manifest.to_dict()

        revoked = self.revoked


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "digest": digest,
            "manifest": manifest,
            "revoked": revoked,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_engine_manifest import VoiceEngineManifest # noqa: PLC0415
        d = dict(src_dict)
        digest = d.pop("digest")

        manifest = VoiceEngineManifest.from_dict(d.pop("manifest"))




        revoked = d.pop("revoked")

        voice_engine_release = cls(
            digest=digest,
            manifest=manifest,
            revoked=revoked,
        )

        return voice_engine_release
