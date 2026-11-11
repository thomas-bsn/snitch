from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceRecomputeReport")



@_attrs_define
class VoiceRecomputeReport:
    """
        Attributes:
            fingerprints (int): Empreintes du recalcul.
            pending (int): Empreintes en attente de leur tâche.
            recomputed (int): Empreintes remplacées par leur recalcul sur la Parole nette.
            recovered (int): Recalculées rejetées avant, comparables désormais.
            recovered_contributive (int): Recalculées non contributives avant, contributives désormais.
            still_rejected (int): Recalculées rejetées avant et après.
            lost (int): Recalculées comparables avant, plus maintenant.
            errors (int): Tâche terminée sans nouvelle empreinte : l'ancienne reste en service.
            cancelled (int): Tâche annulée : l'ancienne empreinte reste en service.
            obsolete (int): Empreinte d'origine remplacée ou supprimée hors du recalcul (nouvelle diarisation, suppression).
            missing_sources (int): Source audio absente : l'empreinte requalifiée reste en service.
            archived (int): Locuteur archivé ou sans diarisation courante : non recalculable, l'empreinte reste en service.
     """

    fingerprints: int
    pending: int
    recomputed: int
    recovered: int
    recovered_contributive: int
    still_rejected: int
    lost: int
    errors: int
    cancelled: int
    obsolete: int
    missing_sources: int
    archived: int





    def to_dict(self) -> dict[str, Any]:
        fingerprints = self.fingerprints

        pending = self.pending

        recomputed = self.recomputed

        recovered = self.recovered

        recovered_contributive = self.recovered_contributive

        still_rejected = self.still_rejected

        lost = self.lost

        errors = self.errors

        cancelled = self.cancelled

        obsolete = self.obsolete

        missing_sources = self.missing_sources

        archived = self.archived


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fingerprints": fingerprints,
            "pending": pending,
            "recomputed": recomputed,
            "recovered": recovered,
            "recovered_contributive": recovered_contributive,
            "still_rejected": still_rejected,
            "lost": lost,
            "errors": errors,
            "cancelled": cancelled,
            "obsolete": obsolete,
            "missing_sources": missing_sources,
            "archived": archived,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fingerprints = d.pop("fingerprints")

        pending = d.pop("pending")

        recomputed = d.pop("recomputed")

        recovered = d.pop("recovered")

        recovered_contributive = d.pop("recovered_contributive")

        still_rejected = d.pop("still_rejected")

        lost = d.pop("lost")

        errors = d.pop("errors")

        cancelled = d.pop("cancelled")

        obsolete = d.pop("obsolete")

        missing_sources = d.pop("missing_sources")

        archived = d.pop("archived")

        voice_recompute_report = cls(
            fingerprints=fingerprints,
            pending=pending,
            recomputed=recomputed,
            recovered=recovered,
            recovered_contributive=recovered_contributive,
            still_rejected=still_rejected,
            lost=lost,
            errors=errors,
            cancelled=cancelled,
            obsolete=obsolete,
            missing_sources=missing_sources,
            archived=archived,
        )

        return voice_recompute_report
