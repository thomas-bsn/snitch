from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.voice_accuracy_criteria import VoiceAccuracyCriteria
  from ..models.voice_accuracy_evaluation import VoiceAccuracyEvaluation
  from ..models.voice_accuracy_library import VoiceAccuracyLibrary
  from ..models.voice_accuracy_person import VoiceAccuracyPerson
  from ..models.voice_automation_event import VoiceAutomationEvent





T = TypeVar("T", bound="VoiceAccuracy")



@_attrs_define
class VoiceAccuracy:
    """
        Attributes:
            measured_at (datetime.datetime):
            auto_enabled (bool):
            automatic_threshold (float):
            precision_threshold (float):
            min_confirmations (int):
            evaluation (VoiceAccuracyEvaluation):
            persons (list[VoiceAccuracyPerson]):
            library (VoiceAccuracyLibrary):
            criteria (VoiceAccuracyCriteria):
            activation_proposed (bool): Les critères sont atteints et l'automatisation est désactivée : Snitch propose
                l'activation à l'administrateur.
            events (list[VoiceAutomationEvent]): Historique des activations et désactivations, du plus récent au plus
                ancien.
     """

    measured_at: datetime.datetime
    auto_enabled: bool
    automatic_threshold: float
    precision_threshold: float
    min_confirmations: int
    evaluation: VoiceAccuracyEvaluation
    persons: list[VoiceAccuracyPerson]
    library: VoiceAccuracyLibrary
    criteria: VoiceAccuracyCriteria
    activation_proposed: bool
    events: list[VoiceAutomationEvent]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_accuracy_criteria import VoiceAccuracyCriteria # noqa: PLC0415
        from ..models.voice_accuracy_evaluation import VoiceAccuracyEvaluation # noqa: PLC0415
        from ..models.voice_accuracy_library import VoiceAccuracyLibrary # noqa: PLC0415
        from ..models.voice_accuracy_person import VoiceAccuracyPerson # noqa: PLC0415
        from ..models.voice_automation_event import VoiceAutomationEvent # noqa: PLC0415
        measured_at = self.measured_at.isoformat()

        auto_enabled = self.auto_enabled

        automatic_threshold = self.automatic_threshold

        precision_threshold = self.precision_threshold

        min_confirmations = self.min_confirmations

        evaluation = self.evaluation.to_dict()

        persons = []
        for persons_item_data in self.persons:
            persons_item = persons_item_data.to_dict()
            persons.append(persons_item)



        library = self.library.to_dict()

        criteria = self.criteria.to_dict()

        activation_proposed = self.activation_proposed

        events = []
        for events_item_data in self.events:
            events_item = events_item_data.to_dict()
            events.append(events_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "measured_at": measured_at,
            "auto_enabled": auto_enabled,
            "automatic_threshold": automatic_threshold,
            "precision_threshold": precision_threshold,
            "min_confirmations": min_confirmations,
            "evaluation": evaluation,
            "persons": persons,
            "library": library,
            "criteria": criteria,
            "activation_proposed": activation_proposed,
            "events": events,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_accuracy_criteria import VoiceAccuracyCriteria # noqa: PLC0415
        from ..models.voice_accuracy_evaluation import VoiceAccuracyEvaluation # noqa: PLC0415
        from ..models.voice_accuracy_library import VoiceAccuracyLibrary # noqa: PLC0415
        from ..models.voice_accuracy_person import VoiceAccuracyPerson # noqa: PLC0415
        from ..models.voice_automation_event import VoiceAutomationEvent # noqa: PLC0415
        d = dict(src_dict)
        measured_at = datetime.datetime.fromisoformat(d.pop("measured_at"))




        auto_enabled = d.pop("auto_enabled")

        automatic_threshold = d.pop("automatic_threshold")

        precision_threshold = d.pop("precision_threshold")

        min_confirmations = d.pop("min_confirmations")

        evaluation = VoiceAccuracyEvaluation.from_dict(d.pop("evaluation"))




        persons = []
        _persons = d.pop("persons")
        for persons_item_data in (_persons):
            persons_item = VoiceAccuracyPerson.from_dict(persons_item_data)



            persons.append(persons_item)


        library = VoiceAccuracyLibrary.from_dict(d.pop("library"))




        criteria = VoiceAccuracyCriteria.from_dict(d.pop("criteria"))




        activation_proposed = d.pop("activation_proposed")

        events = []
        _events = d.pop("events")
        for events_item_data in (_events):
            events_item = VoiceAutomationEvent.from_dict(events_item_data)



            events.append(events_item)


        voice_accuracy = cls(
            measured_at=measured_at,
            auto_enabled=auto_enabled,
            automatic_threshold=automatic_threshold,
            precision_threshold=precision_threshold,
            min_confirmations=min_confirmations,
            evaluation=evaluation,
            persons=persons,
            library=library,
            criteria=criteria,
            activation_proposed=activation_proposed,
            events=events,
        )


        voice_accuracy.additional_properties = d
        return voice_accuracy

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
