from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.person_link_request_status import PersonLinkRequestStatus
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.person_link_account import PersonLinkAccount
  from ..models.person_ref import PersonRef





T = TypeVar("T", bound="PersonLinkRequest")



@_attrs_define
class PersonLinkRequest:
    """
        Attributes:
            id (int):
            person (PersonRef):
            account (PersonLinkAccount):
            status (PersonLinkRequestStatus):
            message (str):
            created_at (datetime.datetime):
            decided_at (datetime.datetime | None):
            decided_by (None | PersonLinkAccount): Administrateur qui a décidé, le cas échéant.
     """

    id: int
    person: PersonRef
    account: PersonLinkAccount
    status: PersonLinkRequestStatus
    message: str
    created_at: datetime.datetime
    decided_at: datetime.datetime | None
    decided_by: None | PersonLinkAccount
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.person_link_account import PersonLinkAccount # noqa: PLC0415
        from ..models.person_ref import PersonRef # noqa: PLC0415
        id = self.id

        person = self.person.to_dict()

        account = self.account.to_dict()

        status = self.status.value

        message = self.message

        created_at = self.created_at.isoformat()

        decided_at: None | str
        if isinstance(self.decided_at, datetime.datetime):
            decided_at = self.decided_at.isoformat()
        else:
            decided_at = self.decided_at

        decided_by: dict[str, Any] | None
        if isinstance(self.decided_by, PersonLinkAccount):
            decided_by = self.decided_by.to_dict()
        else:
            decided_by = self.decided_by


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "person": person,
            "account": account,
            "status": status,
            "message": message,
            "created_at": created_at,
            "decided_at": decided_at,
            "decided_by": decided_by,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.person_link_account import PersonLinkAccount # noqa: PLC0415
        from ..models.person_ref import PersonRef # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        person = PersonRef.from_dict(d.pop("person"))




        account = PersonLinkAccount.from_dict(d.pop("account"))




        status = PersonLinkRequestStatus(d.pop("status"))




        message = d.pop("message")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        def _parse_decided_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                decided_at_type_0 = datetime.datetime.fromisoformat(data)



                return decided_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        decided_at = _parse_decided_at(d.pop("decided_at"))


        def _parse_decided_by(data: object) -> None | PersonLinkAccount:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                decided_by_type_1 = PersonLinkAccount.from_dict(data)



                return decided_by_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | PersonLinkAccount, data)

        decided_by = _parse_decided_by(d.pop("decided_by"))


        person_link_request = cls(
            id=id,
            person=person,
            account=account,
            status=status,
            message=message,
            created_at=created_at,
            decided_at=decided_at,
            decided_by=decided_by,
        )


        person_link_request.additional_properties = d
        return person_link_request

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
