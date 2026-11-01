from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_pipeline_provider import VoicePipelineProvider
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.voice_pipeline_fake import VoicePipelineFake





T = TypeVar("T", bound="VoicePipeline")



@_attrs_define
class VoicePipeline:
    """
        Attributes:
            provider (VoicePipelineProvider):
            model (str):
            model_version (str):
            min_fingerprint_seconds (float):
            min_quality (float):
            fake (VoicePipelineFake | Unset): Réglages du fournisseur factice déterministe, réservés aux tests.
     """

    provider: VoicePipelineProvider
    model: str
    model_version: str
    min_fingerprint_seconds: float
    min_quality: float
    fake: VoicePipelineFake | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_pipeline_fake import VoicePipelineFake # noqa: PLC0415
        provider = self.provider.value

        model = self.model

        model_version = self.model_version

        min_fingerprint_seconds = self.min_fingerprint_seconds

        min_quality = self.min_quality

        fake: dict[str, Any] | Unset = UNSET
        if not isinstance(self.fake, Unset):
            fake = self.fake.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "provider": provider,
            "model": model,
            "model_version": model_version,
            "min_fingerprint_seconds": min_fingerprint_seconds,
            "min_quality": min_quality,
        })
        if fake is not UNSET:
            field_dict["fake"] = fake

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_pipeline_fake import VoicePipelineFake # noqa: PLC0415
        d = dict(src_dict)
        provider = VoicePipelineProvider(d.pop("provider"))




        model = d.pop("model")

        model_version = d.pop("model_version")

        min_fingerprint_seconds = d.pop("min_fingerprint_seconds")

        min_quality = d.pop("min_quality")

        _fake = d.pop("fake", UNSET)
        fake: VoicePipelineFake | Unset
        if isinstance(_fake,  Unset):
            fake = UNSET
        else:
            fake = VoicePipelineFake.from_dict(_fake)




        voice_pipeline = cls(
            provider=provider,
            model=model,
            model_version=model_version,
            min_fingerprint_seconds=min_fingerprint_seconds,
            min_quality=min_quality,
            fake=fake,
        )

        return voice_pipeline
