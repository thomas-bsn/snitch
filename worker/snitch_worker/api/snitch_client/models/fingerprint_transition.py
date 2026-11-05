from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.fingerprint_transition_phase import FingerprintTransitionPhase
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="FingerprintTransition")



@_attrs_define
class FingerprintTransition:
    """
        Attributes:
            id (int):
            from_model_id (str):
            to_model_id (str):
            phase (FingerprintTransitionPhase):
            announced_references (int):
            references_total (int):
            references_done (int):
            references_remaining (int):
            jobs_active (int):
            jobs_failed (int):
            catch_up_remaining (int):
            workers_ready (int):
            author_name (str):
            created_at (datetime.datetime):
            switched_at (datetime.datetime | Unset):
     """

    id: int
    from_model_id: str
    to_model_id: str
    phase: FingerprintTransitionPhase
    announced_references: int
    references_total: int
    references_done: int
    references_remaining: int
    jobs_active: int
    jobs_failed: int
    catch_up_remaining: int
    workers_ready: int
    author_name: str
    created_at: datetime.datetime
    switched_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        from_model_id = self.from_model_id

        to_model_id = self.to_model_id

        phase = self.phase.value

        announced_references = self.announced_references

        references_total = self.references_total

        references_done = self.references_done

        references_remaining = self.references_remaining

        jobs_active = self.jobs_active

        jobs_failed = self.jobs_failed

        catch_up_remaining = self.catch_up_remaining

        workers_ready = self.workers_ready

        author_name = self.author_name

        created_at = self.created_at.isoformat()

        switched_at: str | Unset = UNSET
        if not isinstance(self.switched_at, Unset):
            switched_at = self.switched_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "from_model_id": from_model_id,
            "to_model_id": to_model_id,
            "phase": phase,
            "announced_references": announced_references,
            "references_total": references_total,
            "references_done": references_done,
            "references_remaining": references_remaining,
            "jobs_active": jobs_active,
            "jobs_failed": jobs_failed,
            "catch_up_remaining": catch_up_remaining,
            "workers_ready": workers_ready,
            "author_name": author_name,
            "created_at": created_at,
        })
        if switched_at is not UNSET:
            field_dict["switched_at"] = switched_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        from_model_id = d.pop("from_model_id")

        to_model_id = d.pop("to_model_id")

        phase = FingerprintTransitionPhase(d.pop("phase"))




        announced_references = d.pop("announced_references")

        references_total = d.pop("references_total")

        references_done = d.pop("references_done")

        references_remaining = d.pop("references_remaining")

        jobs_active = d.pop("jobs_active")

        jobs_failed = d.pop("jobs_failed")

        catch_up_remaining = d.pop("catch_up_remaining")

        workers_ready = d.pop("workers_ready")

        author_name = d.pop("author_name")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        _switched_at = d.pop("switched_at", UNSET)
        switched_at: datetime.datetime | Unset
        if isinstance(_switched_at,  Unset):
            switched_at = UNSET
        else:
            switched_at = datetime.datetime.fromisoformat(_switched_at)




        fingerprint_transition = cls(
            id=id,
            from_model_id=from_model_id,
            to_model_id=to_model_id,
            phase=phase,
            announced_references=announced_references,
            references_total=references_total,
            references_done=references_done,
            references_remaining=references_remaining,
            jobs_active=jobs_active,
            jobs_failed=jobs_failed,
            catch_up_remaining=catch_up_remaining,
            workers_ready=workers_ready,
            author_name=author_name,
            created_at=created_at,
            switched_at=switched_at,
        )

        return fingerprint_transition
