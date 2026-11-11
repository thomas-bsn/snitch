from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceRecomputePreview")



@_attrs_define
class VoiceRecomputePreview:
    """
        Attributes:
            fingerprints (int): Empreintes de Locuteur local de production au vecteur d'origine, dans le groupe du modèle
                actif.
            recomputable (int): Dont recalculables.
            jobs (int): Tâches d'empreinte à créer, une par diarisation.
            clips (int): Clips concernés.
            missing_sources (int): Empreintes dont la source audio est absente : signalées, laissées en service.
            archived (int): Empreintes de Locuteurs archivés sans diarisation courante : non recalculables.
            references (int): Empreintes d'un échantillon validé actif.
            transition_open (bool): Une transition de modèle d'empreinte est ouverte : le recalcul est refusé.
            running (bool): Un recalcul est déjà en cours.
     """

    fingerprints: int
    recomputable: int
    jobs: int
    clips: int
    missing_sources: int
    archived: int
    references: int
    transition_open: bool
    running: bool





    def to_dict(self) -> dict[str, Any]:
        fingerprints = self.fingerprints

        recomputable = self.recomputable

        jobs = self.jobs

        clips = self.clips

        missing_sources = self.missing_sources

        archived = self.archived

        references = self.references

        transition_open = self.transition_open

        running = self.running


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fingerprints": fingerprints,
            "recomputable": recomputable,
            "jobs": jobs,
            "clips": clips,
            "missing_sources": missing_sources,
            "archived": archived,
            "references": references,
            "transition_open": transition_open,
            "running": running,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fingerprints = d.pop("fingerprints")

        recomputable = d.pop("recomputable")

        jobs = d.pop("jobs")

        clips = d.pop("clips")

        missing_sources = d.pop("missing_sources")

        archived = d.pop("archived")

        references = d.pop("references")

        transition_open = d.pop("transition_open")

        running = d.pop("running")

        voice_recompute_preview = cls(
            fingerprints=fingerprints,
            recomputable=recomputable,
            jobs=jobs,
            clips=clips,
            missing_sources=missing_sources,
            archived=archived,
            references=references,
            transition_open=transition_open,
            running=running,
        )

        return voice_recompute_preview
