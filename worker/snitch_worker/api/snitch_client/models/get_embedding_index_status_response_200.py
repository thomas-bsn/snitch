from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="GetEmbeddingIndexStatusResponse200")



@_attrs_define
class GetEmbeddingIndexStatusResponse200:
    """
        Attributes:
            configured (bool):
            provider (str):
            model (str):
            dimensions (int):
            state (str):
            pending (int):
            done (int):
            failed (int):
            indexed_segments (int): Segments de l’index actif.
            reindex_pending (int): Segments corrigés dont le vecteur n’est pas encore recalculé (actuels, non obsolètes).
            reindex_failed (int): Parmi eux, ceux dont les tentatives sont épuisées : une relance de l’indexation (backfill)
                leur donne une nouvelle chance.
            error (str | Unset):
            candidate_model (str | Unset):
            candidate_state (str | Unset):
            candidate_error (str | Unset):
            candidate_total (int | Unset):
            candidate_done (int | Unset):
            candidate_pending (int | Unset):
            candidate_failed (int | Unset):
     """

    configured: bool
    provider: str
    model: str
    dimensions: int
    state: str
    pending: int
    done: int
    failed: int
    indexed_segments: int
    reindex_pending: int
    reindex_failed: int
    error: str | Unset = UNSET
    candidate_model: str | Unset = UNSET
    candidate_state: str | Unset = UNSET
    candidate_error: str | Unset = UNSET
    candidate_total: int | Unset = UNSET
    candidate_done: int | Unset = UNSET
    candidate_pending: int | Unset = UNSET
    candidate_failed: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        configured = self.configured

        provider = self.provider

        model = self.model

        dimensions = self.dimensions

        state = self.state

        pending = self.pending

        done = self.done

        failed = self.failed

        indexed_segments = self.indexed_segments

        reindex_pending = self.reindex_pending

        reindex_failed = self.reindex_failed

        error = self.error

        candidate_model = self.candidate_model

        candidate_state = self.candidate_state

        candidate_error = self.candidate_error

        candidate_total = self.candidate_total

        candidate_done = self.candidate_done

        candidate_pending = self.candidate_pending

        candidate_failed = self.candidate_failed


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "configured": configured,
            "provider": provider,
            "model": model,
            "dimensions": dimensions,
            "state": state,
            "pending": pending,
            "done": done,
            "failed": failed,
            "indexed_segments": indexed_segments,
            "reindex_pending": reindex_pending,
            "reindex_failed": reindex_failed,
        })
        if error is not UNSET:
            field_dict["error"] = error
        if candidate_model is not UNSET:
            field_dict["candidate_model"] = candidate_model
        if candidate_state is not UNSET:
            field_dict["candidate_state"] = candidate_state
        if candidate_error is not UNSET:
            field_dict["candidate_error"] = candidate_error
        if candidate_total is not UNSET:
            field_dict["candidate_total"] = candidate_total
        if candidate_done is not UNSET:
            field_dict["candidate_done"] = candidate_done
        if candidate_pending is not UNSET:
            field_dict["candidate_pending"] = candidate_pending
        if candidate_failed is not UNSET:
            field_dict["candidate_failed"] = candidate_failed

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        configured = d.pop("configured")

        provider = d.pop("provider")

        model = d.pop("model")

        dimensions = d.pop("dimensions")

        state = d.pop("state")

        pending = d.pop("pending")

        done = d.pop("done")

        failed = d.pop("failed")

        indexed_segments = d.pop("indexed_segments")

        reindex_pending = d.pop("reindex_pending")

        reindex_failed = d.pop("reindex_failed")

        error = d.pop("error", UNSET)

        candidate_model = d.pop("candidate_model", UNSET)

        candidate_state = d.pop("candidate_state", UNSET)

        candidate_error = d.pop("candidate_error", UNSET)

        candidate_total = d.pop("candidate_total", UNSET)

        candidate_done = d.pop("candidate_done", UNSET)

        candidate_pending = d.pop("candidate_pending", UNSET)

        candidate_failed = d.pop("candidate_failed", UNSET)

        get_embedding_index_status_response_200 = cls(
            configured=configured,
            provider=provider,
            model=model,
            dimensions=dimensions,
            state=state,
            pending=pending,
            done=done,
            failed=failed,
            indexed_segments=indexed_segments,
            reindex_pending=reindex_pending,
            reindex_failed=reindex_failed,
            error=error,
            candidate_model=candidate_model,
            candidate_state=candidate_state,
            candidate_error=candidate_error,
            candidate_total=candidate_total,
            candidate_done=candidate_done,
            candidate_pending=candidate_pending,
            candidate_failed=candidate_failed,
        )


        get_embedding_index_status_response_200.additional_properties = d
        return get_embedding_index_status_response_200

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
