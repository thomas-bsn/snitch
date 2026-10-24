from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.clip_page import ClipPage
from ...models.error import Error
from ...models.list_clips_status import ListClipsStatus
from ...models.list_clips_visibility import ListClipsVisibility
from ...types import UNSET, Unset
from typing import cast
import datetime



def _get_kwargs(
    *,
    page: int | Unset = UNSET,
    page_size: int | Unset = UNSET,
    visibility: ListClipsVisibility | Unset = UNSET,
    status: ListClipsStatus | Unset = UNSET,
    owner_id: int | Unset = UNSET,
    source_id: int | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    to: datetime.date | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["page"] = page

    params["page_size"] = page_size

    json_visibility: str | Unset = UNSET
    if not isinstance(visibility, Unset):
        json_visibility = visibility.value

    params["visibility"] = json_visibility

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    params["owner_id"] = owner_id

    params["source_id"] = source_id

    json_from_: str | Unset = UNSET
    if not isinstance(from_, Unset):
        json_from_ = from_.isoformat()
    params["from"] = json_from_

    json_to: str | Unset = UNSET
    if not isinstance(to, Unset):
        json_to = to.isoformat()
    params["to"] = json_to


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/clips",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ClipPage | Error:
    if response.status_code == 200:
        response_200 = ClipPage.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ClipPage | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    page: int | Unset = UNSET,
    page_size: int | Unset = UNSET,
    visibility: ListClipsVisibility | Unset = UNSET,
    status: ListClipsStatus | Unset = UNSET,
    owner_id: int | Unset = UNSET,
    source_id: int | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    to: datetime.date | Unset = UNSET,

) -> Response[ClipPage | Error]:
    """  Session requise. Les clips privés hors propriétaire/admin sont exclus avant filtres et pagination.

    Args:
        page (int | Unset):
        page_size (int | Unset):
        visibility (ListClipsVisibility | Unset):
        status (ListClipsStatus | Unset):
        owner_id (int | Unset):
        source_id (int | Unset):
        from_ (datetime.date | Unset):
        to (datetime.date | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClipPage | Error]
     """


    kwargs = _get_kwargs(
        page=page,
page_size=page_size,
visibility=visibility,
status=status,
owner_id=owner_id,
source_id=source_id,
from_=from_,
to=to,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    page: int | Unset = UNSET,
    page_size: int | Unset = UNSET,
    visibility: ListClipsVisibility | Unset = UNSET,
    status: ListClipsStatus | Unset = UNSET,
    owner_id: int | Unset = UNSET,
    source_id: int | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    to: datetime.date | Unset = UNSET,

) -> ClipPage | Error | None:
    """  Session requise. Les clips privés hors propriétaire/admin sont exclus avant filtres et pagination.

    Args:
        page (int | Unset):
        page_size (int | Unset):
        visibility (ListClipsVisibility | Unset):
        status (ListClipsStatus | Unset):
        owner_id (int | Unset):
        source_id (int | Unset):
        from_ (datetime.date | Unset):
        to (datetime.date | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClipPage | Error
     """


    return sync_detailed(
        client=client,
page=page,
page_size=page_size,
visibility=visibility,
status=status,
owner_id=owner_id,
source_id=source_id,
from_=from_,
to=to,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    page: int | Unset = UNSET,
    page_size: int | Unset = UNSET,
    visibility: ListClipsVisibility | Unset = UNSET,
    status: ListClipsStatus | Unset = UNSET,
    owner_id: int | Unset = UNSET,
    source_id: int | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    to: datetime.date | Unset = UNSET,

) -> Response[ClipPage | Error]:
    """  Session requise. Les clips privés hors propriétaire/admin sont exclus avant filtres et pagination.

    Args:
        page (int | Unset):
        page_size (int | Unset):
        visibility (ListClipsVisibility | Unset):
        status (ListClipsStatus | Unset):
        owner_id (int | Unset):
        source_id (int | Unset):
        from_ (datetime.date | Unset):
        to (datetime.date | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClipPage | Error]
     """


    kwargs = _get_kwargs(
        page=page,
page_size=page_size,
visibility=visibility,
status=status,
owner_id=owner_id,
source_id=source_id,
from_=from_,
to=to,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    page: int | Unset = UNSET,
    page_size: int | Unset = UNSET,
    visibility: ListClipsVisibility | Unset = UNSET,
    status: ListClipsStatus | Unset = UNSET,
    owner_id: int | Unset = UNSET,
    source_id: int | Unset = UNSET,
    from_: datetime.date | Unset = UNSET,
    to: datetime.date | Unset = UNSET,

) -> ClipPage | Error | None:
    """  Session requise. Les clips privés hors propriétaire/admin sont exclus avant filtres et pagination.

    Args:
        page (int | Unset):
        page_size (int | Unset):
        visibility (ListClipsVisibility | Unset):
        status (ListClipsStatus | Unset):
        owner_id (int | Unset):
        source_id (int | Unset):
        from_ (datetime.date | Unset):
        to (datetime.date | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClipPage | Error
     """


    return (await asyncio_detailed(
        client=client,
page=page,
page_size=page_size,
visibility=visibility,
status=status,
owner_id=owner_id,
source_id=source_id,
from_=from_,
to=to,

    )).parsed
