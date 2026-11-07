from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.audio_loss_preview import AudioLossPreview





T = TypeVar("T", bound="AudioPublicationPreview")



@_attrs_define
class AudioPublicationPreview:
    """ Pertes de la publication d'un plan et état de la confirmation de l'administrateur.

        Attributes:
            plan_id (int):
            guard (str): Empreinte de l'historique des corrections et validations vu par l'aperçu.
            losses (AudioLossPreview):
            confirmed (bool): Les pertes sont acceptées pour cette empreinte.
            confirmation_current (bool): La confirmation enregistrée porte sur l'empreinte et la révision actuelles.
     """

    plan_id: int
    guard: str
    losses: AudioLossPreview
    confirmed: bool
    confirmation_current: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_loss_preview import AudioLossPreview # noqa: PLC0415
        plan_id = self.plan_id

        guard = self.guard

        losses = self.losses.to_dict()

        confirmed = self.confirmed

        confirmation_current = self.confirmation_current


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "plan_id": plan_id,
            "guard": guard,
            "losses": losses,
            "confirmed": confirmed,
            "confirmation_current": confirmation_current,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_loss_preview import AudioLossPreview # noqa: PLC0415
        d = dict(src_dict)
        plan_id = d.pop("plan_id")

        guard = d.pop("guard")

        losses = AudioLossPreview.from_dict(d.pop("losses"))




        confirmed = d.pop("confirmed")

        confirmation_current = d.pop("confirmation_current")

        audio_publication_preview = cls(
            plan_id=plan_id,
            guard=guard,
            losses=losses,
            confirmed=confirmed,
            confirmation_current=confirmation_current,
        )


        audio_publication_preview.additional_properties = d
        return audio_publication_preview

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
