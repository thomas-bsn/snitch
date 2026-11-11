from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_eligibility_outcome import VoiceEligibilityOutcome
  from ..models.voice_eligibility_person_preview import VoiceEligibilityPersonPreview
  from ..models.voice_eligibility_scenario import VoiceEligibilityScenario





T = TypeVar("T", bound="VoiceEligibilityGroupPreview")



@_attrs_define
class VoiceEligibilityGroupPreview:
    """ Un groupe d'empreintes de production (modèle, version, dimensions).

        Attributes:
            model (str):
            model_version (str):
            dimensions (int):
            missing_sources (int): Empreintes dont le clip n'a plus de source audio connue en base (aucun média lu).
            estimated (int): Empreintes de Locuteur local archivé sans tours : Parole nette estimée par la durée de
                l'empreinte.
            original_vectors (int): Empreintes dont le vecteur précède la Parole nette (sans mesures du Worker) : vecteur
                d'origine, requalifié sans recalcul.
            overlap_share (float | None): Part de la parole retirée par les Chevauchements, pondérée par le temps de parole,
                sur les empreintes dont le Chevauchement est connu ; null sans aucune.
            current (VoiceEligibilityOutcome): Empreintes de Locuteur local de production par éligibilité, et Personnes qui
                ont une Référence.
            candidate (VoiceEligibilityScenario): Éligibilité qu'une requalification sous cette règle écrirait, comparée à
                l'état enregistré.
            scenarios (list[VoiceEligibilityScenario]): Une par durée minimale, appliquée aux deux usages, avec la part
                d'écrêtage examinée.
            persons (list[VoiceEligibilityPersonPreview]):
     """

    model: str
    model_version: str
    dimensions: int
    missing_sources: int
    estimated: int
    original_vectors: int
    overlap_share: float | None
    current: VoiceEligibilityOutcome
    candidate: VoiceEligibilityScenario
    scenarios: list[VoiceEligibilityScenario]
    persons: list[VoiceEligibilityPersonPreview]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_eligibility_outcome import VoiceEligibilityOutcome # noqa: PLC0415
        from ..models.voice_eligibility_person_preview import VoiceEligibilityPersonPreview # noqa: PLC0415
        from ..models.voice_eligibility_scenario import VoiceEligibilityScenario # noqa: PLC0415
        model = self.model

        model_version = self.model_version

        dimensions = self.dimensions

        missing_sources = self.missing_sources

        estimated = self.estimated

        original_vectors = self.original_vectors

        overlap_share: float | None
        overlap_share = self.overlap_share

        current = self.current.to_dict()

        candidate = self.candidate.to_dict()

        scenarios = []
        for scenarios_item_data in self.scenarios:
            scenarios_item = scenarios_item_data.to_dict()
            scenarios.append(scenarios_item)



        persons = []
        for persons_item_data in self.persons:
            persons_item = persons_item_data.to_dict()
            persons.append(persons_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "model": model,
            "model_version": model_version,
            "dimensions": dimensions,
            "missing_sources": missing_sources,
            "estimated": estimated,
            "original_vectors": original_vectors,
            "overlap_share": overlap_share,
            "current": current,
            "candidate": candidate,
            "scenarios": scenarios,
            "persons": persons,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_eligibility_outcome import VoiceEligibilityOutcome # noqa: PLC0415
        from ..models.voice_eligibility_person_preview import VoiceEligibilityPersonPreview # noqa: PLC0415
        from ..models.voice_eligibility_scenario import VoiceEligibilityScenario # noqa: PLC0415
        d = dict(src_dict)
        model = d.pop("model")

        model_version = d.pop("model_version")

        dimensions = d.pop("dimensions")

        missing_sources = d.pop("missing_sources")

        estimated = d.pop("estimated")

        original_vectors = d.pop("original_vectors")

        def _parse_overlap_share(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        overlap_share = _parse_overlap_share(d.pop("overlap_share"))


        current = VoiceEligibilityOutcome.from_dict(d.pop("current"))




        candidate = VoiceEligibilityScenario.from_dict(d.pop("candidate"))




        scenarios = []
        _scenarios = d.pop("scenarios")
        for scenarios_item_data in (_scenarios):
            scenarios_item = VoiceEligibilityScenario.from_dict(scenarios_item_data)



            scenarios.append(scenarios_item)


        persons = []
        _persons = d.pop("persons")
        for persons_item_data in (_persons):
            persons_item = VoiceEligibilityPersonPreview.from_dict(persons_item_data)



            persons.append(persons_item)


        voice_eligibility_group_preview = cls(
            model=model,
            model_version=model_version,
            dimensions=dimensions,
            missing_sources=missing_sources,
            estimated=estimated,
            original_vectors=original_vectors,
            overlap_share=overlap_share,
            current=current,
            candidate=candidate,
            scenarios=scenarios,
            persons=persons,
        )


        voice_eligibility_group_preview.additional_properties = d
        return voice_eligibility_group_preview

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
