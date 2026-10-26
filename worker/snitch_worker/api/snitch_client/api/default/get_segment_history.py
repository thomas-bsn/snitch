from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.segment_history import SegmentHistory
from typing import cast



def _get_kwargs(
    id: int,
    index: int,

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/clips/{id}/segments/{index}/history".format(id=quote(str(id), safe=""),index=quote(str(index), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | SegmentHistory:
    if response.status_code == 200:
        response_200 = SegmentHistory.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | SegmentHistory]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    index: int,
    *,
    client: AuthenticatedClient,

) -> Response[Error | SegmentHistory]:
    """  Utilisateurs connectés qui peuvent voir le clip. Les entrées sont immuables : ancienne et nouvelle
    valeur effectives, auteur, date et motif. Seul l'historique du segment de la transcription visible
    est lu.

    Args:
        id (int):
        index (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SegmentHistory]
     """


    kwargs = _get_kwargs(
        id=id,
index=index,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    index: int,
    *,
    client: AuthenticatedClient,

) -> Error | SegmentHistory | None:
    """  Utilisateurs connectés qui peuvent voir le clip. Les entrées sont immuables : ancienne et nouvelle
    valeur effectives, auteur, date et motif. Seul l'historique du segment de la transcription visible
    est lu.

    Args:
        id (int):
        index (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SegmentHistory
     """


    return sync_detailed(
        id=id,
index=index,
client=client,

    ).parsed

async def asyncio_detailed(
    id: int,
    index: int,
    *,
    client: AuthenticatedClient,

) -> Response[Error | SegmentHistory]:
    """  Utilisateurs connectés qui peuvent voir le clip. Les entrées sont immuables : ancienne et nouvelle
    valeur effectives, auteur, date et motif. Seul l'historique du segment de la transcription visible
    est lu.

    Args:
        id (int):
        index (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SegmentHistory]
     """


    kwargs = _get_kwargs(
        id=id,
index=index,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    index: int,
    *,
    client: AuthenticatedClient,

) -> Error | SegmentHistory | None:
    """  Utilisateurs connectés qui peuvent voir le clip. Les entrées sont immuables : ancienne et nouvelle
    valeur effectives, auteur, date et motif. Seul l'historique du segment de la transcription visible
    est lu.

    Args:
        id (int):
        index (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SegmentHistory
     """


    return (await asyncio_detailed(
        id=id,
index=index,
client=client,

    )).parsed
