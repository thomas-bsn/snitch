from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.person_reference_basis import PersonReferenceBasis
from ..models.person_reference_level import PersonReferenceLevel
from typing import cast

if TYPE_CHECKING:
  from ..models.person_reference_discard import PersonReferenceDiscard





T = TypeVar("T", bound="PersonReference")



@_attrs_define
class PersonReference:
    """ La Référence de la Personne telle que le rapprochement l'utilise, dans le groupe d'empreinte (modèle, version,
    dimensions) actif, et son niveau. Le niveau est calculé par le serveur sur toutes les Personnes du groupe, quels que
    soient le lecteur et la page lue.

        Attributes:
            level (PersonReferenceLevel): none sans échantillon contributif ; weak sur un seul clip ; sinon, à partir de 3
                Personnes avec Référence dans le groupe, good à partir de la médiane de leur Parole nette et weak en dessous ;
                avec moins de 3 Références, good.
            basis (PersonReferenceBasis): Pourquoi la Référence a ce niveau.
            contributive_samples (int): Échantillons validés actifs qui entrent dans la Référence (empreinte contributive du
                groupe actif).
            clips (int): Clips distincts de ces échantillons.
            net_speech_seconds (float): Parole nette cumulée de ces échantillons, après plafond par échantillon.
            discarded (list[PersonReferenceDiscard]): Échantillons validés actifs écartés de la Référence, groupés par
                raison.
     """

    level: PersonReferenceLevel
    basis: PersonReferenceBasis
    contributive_samples: int
    clips: int
    net_speech_seconds: float
    discarded: list[PersonReferenceDiscard]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.person_reference_discard import PersonReferenceDiscard # noqa: PLC0415
        level = self.level.value

        basis = self.basis.value

        contributive_samples = self.contributive_samples

        clips = self.clips

        net_speech_seconds = self.net_speech_seconds

        discarded = []
        for discarded_item_data in self.discarded:
            discarded_item = discarded_item_data.to_dict()
            discarded.append(discarded_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "level": level,
            "basis": basis,
            "contributive_samples": contributive_samples,
            "clips": clips,
            "net_speech_seconds": net_speech_seconds,
            "discarded": discarded,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.person_reference_discard import PersonReferenceDiscard # noqa: PLC0415
        d = dict(src_dict)
        level = PersonReferenceLevel(d.pop("level"))




        basis = PersonReferenceBasis(d.pop("basis"))




        contributive_samples = d.pop("contributive_samples")

        clips = d.pop("clips")

        net_speech_seconds = d.pop("net_speech_seconds")

        discarded = []
        _discarded = d.pop("discarded")
        for discarded_item_data in (_discarded):
            discarded_item = PersonReferenceDiscard.from_dict(discarded_item_data)



            discarded.append(discarded_item)


        person_reference = cls(
            level=level,
            basis=basis,
            contributive_samples=contributive_samples,
            clips=clips,
            net_speech_seconds=net_speech_seconds,
            discarded=discarded,
        )


        person_reference.additional_properties = d
        return person_reference

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
