from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.clip_step_state import ClipStepState






T = TypeVar("T", bound="ClipProgress")



@_attrs_define
class ClipProgress:
    """ Avancement du clip pour les étapes que le serveur seul connaît. Les étapes de voix se lisent avec GET
    /api/clips/{id}/voice. Un clip est toujours importé.

        Attributes:
            transcription (ClipStepState): État d’une étape du clip : none (rien de planifié), queued (en file), running (en
                cours), done (terminée), failed (échouée), unavailable (étape non configurée sur cette installation).
            indexing (ClipStepState): État d’une étape du clip : none (rien de planifié), queued (en file), running (en
                cours), done (terminée), failed (échouée), unavailable (étape non configurée sur cette installation).
     """

    transcription: ClipStepState
    indexing: ClipStepState
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        transcription = self.transcription.value

        indexing = self.indexing.value


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "transcription": transcription,
            "indexing": indexing,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        transcription = ClipStepState(d.pop("transcription"))




        indexing = ClipStepState(d.pop("indexing"))




        clip_progress = cls(
            transcription=transcription,
            indexing=indexing,
        )


        clip_progress.additional_properties = d
        return clip_progress

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
