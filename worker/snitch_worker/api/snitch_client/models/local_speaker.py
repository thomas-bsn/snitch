from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.decision_level import DecisionLevel
from ..models.local_speaker_disposition import LocalSpeakerDisposition
from ..models.local_speaker_kind import LocalSpeakerKind
from ..models.local_speaker_state import LocalSpeakerState
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.attribution import Attribution
  from ..models.person_ref import PersonRef
  from ..models.voice_fingerprint import VoiceFingerprint





T = TypeVar("T", bound="LocalSpeaker")



@_attrs_define
class LocalSpeaker:
    """
        Attributes:
            id (int):
            label (str):
            kind (LocalSpeakerKind): microphone : piste micro désignée, attribuée sans diarisation (un seul tour couvrant le
                clip, aucune empreinte).
            ordinal (int):
            speaking_seconds (float):
            turn_count (int):
            state (LocalSpeakerState): identified : attribution humaine ou automatique à une Personne. Une suggestion ou une
                voix ambiguë reste unidentified ; une voix inconnue ne bloque rien.
            decision_level (DecisionLevel): human : validation humaine ; automatic, suggestion, ambiguous : issus du
                rapprochement en base (automatic seulement lorsque l'administrateur a activé l'automatisation) ; unknown :
                aucune attribution ou aucune Personne assez proche.
            person (None | PersonRef): Personne de l'attribution effective (validation humaine prioritaire).
            attribution (Attribution | None):
            fingerprints (list[VoiceFingerprint]):
            disposition (LocalSpeakerDisposition | Unset): ignored : voix mise de côté, sans locuteur sur ses segments ;
                random : segments libellés « Random ». Absente pour une voix traitée normalement (ADR 0031).
            refused_persons (list[PersonRef] | Unset): Personnes refusées pour cette voix (« Ce n'est pas X »), que le
                rapprochement ne propose plus ; chacune peut être rétablie par Annuler le refus (ADR 0031).
     """

    id: int
    label: str
    kind: LocalSpeakerKind
    ordinal: int
    speaking_seconds: float
    turn_count: int
    state: LocalSpeakerState
    decision_level: DecisionLevel
    person: None | PersonRef
    attribution: Attribution | None
    fingerprints: list[VoiceFingerprint]
    disposition: LocalSpeakerDisposition | Unset = UNSET
    refused_persons: list[PersonRef] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.attribution import Attribution # noqa: PLC0415
        from ..models.person_ref import PersonRef # noqa: PLC0415
        from ..models.voice_fingerprint import VoiceFingerprint # noqa: PLC0415
        id = self.id

        label = self.label

        kind = self.kind.value

        ordinal = self.ordinal

        speaking_seconds = self.speaking_seconds

        turn_count = self.turn_count

        state = self.state.value

        decision_level = self.decision_level.value

        person: dict[str, Any] | None
        if isinstance(self.person, PersonRef):
            person = self.person.to_dict()
        else:
            person = self.person

        attribution: dict[str, Any] | None
        if isinstance(self.attribution, Attribution):
            attribution = self.attribution.to_dict()
        else:
            attribution = self.attribution

        fingerprints = []
        for fingerprints_item_data in self.fingerprints:
            fingerprints_item = fingerprints_item_data.to_dict()
            fingerprints.append(fingerprints_item)



        disposition: str | Unset = UNSET
        if not isinstance(self.disposition, Unset):
            disposition = self.disposition.value


        refused_persons: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.refused_persons, Unset):
            refused_persons = []
            for refused_persons_item_data in self.refused_persons:
                refused_persons_item = refused_persons_item_data.to_dict()
                refused_persons.append(refused_persons_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "label": label,
            "kind": kind,
            "ordinal": ordinal,
            "speaking_seconds": speaking_seconds,
            "turn_count": turn_count,
            "state": state,
            "decision_level": decision_level,
            "person": person,
            "attribution": attribution,
            "fingerprints": fingerprints,
        })
        if disposition is not UNSET:
            field_dict["disposition"] = disposition
        if refused_persons is not UNSET:
            field_dict["refused_persons"] = refused_persons

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.attribution import Attribution # noqa: PLC0415
        from ..models.person_ref import PersonRef # noqa: PLC0415
        from ..models.voice_fingerprint import VoiceFingerprint # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        label = d.pop("label")

        kind = LocalSpeakerKind(d.pop("kind"))




        ordinal = d.pop("ordinal")

        speaking_seconds = d.pop("speaking_seconds")

        turn_count = d.pop("turn_count")

        state = LocalSpeakerState(d.pop("state"))




        decision_level = DecisionLevel(d.pop("decision_level"))




        def _parse_person(data: object) -> None | PersonRef:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                person_type_1 = PersonRef.from_dict(data)



                return person_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | PersonRef, data)

        person = _parse_person(d.pop("person"))


        def _parse_attribution(data: object) -> Attribution | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                attribution_type_1 = Attribution.from_dict(data)



                return attribution_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Attribution | None, data)

        attribution = _parse_attribution(d.pop("attribution"))


        fingerprints = []
        _fingerprints = d.pop("fingerprints")
        for fingerprints_item_data in (_fingerprints):
            fingerprints_item = VoiceFingerprint.from_dict(fingerprints_item_data)



            fingerprints.append(fingerprints_item)


        _disposition = d.pop("disposition", UNSET)
        disposition: LocalSpeakerDisposition | Unset
        if isinstance(_disposition,  Unset):
            disposition = UNSET
        else:
            disposition = LocalSpeakerDisposition(_disposition)




        _refused_persons = d.pop("refused_persons", UNSET)
        refused_persons: list[PersonRef] | Unset = UNSET
        if _refused_persons is not UNSET:
            refused_persons = []
            for refused_persons_item_data in _refused_persons:
                refused_persons_item = PersonRef.from_dict(refused_persons_item_data)



                refused_persons.append(refused_persons_item)


        local_speaker = cls(
            id=id,
            label=label,
            kind=kind,
            ordinal=ordinal,
            speaking_seconds=speaking_seconds,
            turn_count=turn_count,
            state=state,
            decision_level=decision_level,
            person=person,
            attribution=attribution,
            fingerprints=fingerprints,
            disposition=disposition,
            refused_persons=refused_persons,
        )


        local_speaker.additional_properties = d
        return local_speaker

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
