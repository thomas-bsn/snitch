from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="FingerprintTransitionPreview")



@_attrs_define
class FingerprintTransitionPreview:
    """
        Attributes:
            from_model_id (str):
            to_model_id (str):
            references (int):
            clips (int):
            persons (int):
            other_fingerprints (int):
            validated_attributions (int):
            workers_ready (int):
     """

    from_model_id: str
    to_model_id: str
    references: int
    clips: int
    persons: int
    other_fingerprints: int
    validated_attributions: int
    workers_ready: int





    def to_dict(self) -> dict[str, Any]:
        from_model_id = self.from_model_id

        to_model_id = self.to_model_id

        references = self.references

        clips = self.clips

        persons = self.persons

        other_fingerprints = self.other_fingerprints

        validated_attributions = self.validated_attributions

        workers_ready = self.workers_ready


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "from_model_id": from_model_id,
            "to_model_id": to_model_id,
            "references": references,
            "clips": clips,
            "persons": persons,
            "other_fingerprints": other_fingerprints,
            "validated_attributions": validated_attributions,
            "workers_ready": workers_ready,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        from_model_id = d.pop("from_model_id")

        to_model_id = d.pop("to_model_id")

        references = d.pop("references")

        clips = d.pop("clips")

        persons = d.pop("persons")

        other_fingerprints = d.pop("other_fingerprints")

        validated_attributions = d.pop("validated_attributions")

        workers_ready = d.pop("workers_ready")

        fingerprint_transition_preview = cls(
            from_model_id=from_model_id,
            to_model_id=to_model_id,
            references=references,
            clips=clips,
            persons=persons,
            other_fingerprints=other_fingerprints,
            validated_attributions=validated_attributions,
            workers_ready=workers_ready,
        )

        return fingerprint_transition_preview
