from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="VoiceAccuracyLibrary")



@_attrs_define
class VoiceAccuracyLibrary:
    """
        Attributes:
            speakers_total (int): Locuteurs locaux diarisés (hors piste micro).
            validated (int): Validés par un humain.
            automatic (int): Attribués automatiquement, ou qui le seraient dès l'activation (suggestion au moins égale au
                seuil automatique).
            suggestion (int):
            ambiguous (int):
            unknown (int):
            to_verify (int): Volume de vérification restant : suggestions, ambigus et inconnus.
            coverage (float | None): automatic / locuteurs non validés ; null sans locuteur à attribuer.
            microphone_assigned (int): Pistes micro attribuées à leur propriétaire : comptées à part, indépendantes de
                l'interrupteur.
     """

    speakers_total: int
    validated: int
    automatic: int
    suggestion: int
    ambiguous: int
    unknown: int
    to_verify: int
    coverage: float | None
    microphone_assigned: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        speakers_total = self.speakers_total

        validated = self.validated

        automatic = self.automatic

        suggestion = self.suggestion

        ambiguous = self.ambiguous

        unknown = self.unknown

        to_verify = self.to_verify

        coverage: float | None
        coverage = self.coverage

        microphone_assigned = self.microphone_assigned


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "speakers_total": speakers_total,
            "validated": validated,
            "automatic": automatic,
            "suggestion": suggestion,
            "ambiguous": ambiguous,
            "unknown": unknown,
            "to_verify": to_verify,
            "coverage": coverage,
            "microphone_assigned": microphone_assigned,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        speakers_total = d.pop("speakers_total")

        validated = d.pop("validated")

        automatic = d.pop("automatic")

        suggestion = d.pop("suggestion")

        ambiguous = d.pop("ambiguous")

        unknown = d.pop("unknown")

        to_verify = d.pop("to_verify")

        def _parse_coverage(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        coverage = _parse_coverage(d.pop("coverage"))


        microphone_assigned = d.pop("microphone_assigned")

        voice_accuracy_library = cls(
            speakers_total=speakers_total,
            validated=validated,
            automatic=automatic,
            suggestion=suggestion,
            ambiguous=ambiguous,
            unknown=unknown,
            to_verify=to_verify,
            coverage=coverage,
            microphone_assigned=microphone_assigned,
        )


        voice_accuracy_library.additional_properties = d
        return voice_accuracy_library

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
