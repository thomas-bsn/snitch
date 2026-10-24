from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...types import File, FileTypes
from ...types import UNSET, Unset
from io import BytesIO
from typing import cast



def _get_kwargs(
    id: int,
    *,
    range_: str | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(range_, Unset):
        headers["Range"] = range_







    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/clips/{id}/media".format(id=quote(str(id), safe=""),),
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Error | File:
    if response.status_code == 200:
        response_200 = File(
             payload = BytesIO(response.content)
        )



        return response_200

    if response.status_code == 206:
        response_206 = File(
             payload = BytesIO(response.content)
        )



        return response_206

    if response.status_code == 404:
        response_404 = cast(Any, None)
        return response_404

    if response.status_code == 416:
        response_416 = cast(Any, None)
        return response_416

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | Error | File]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    range_: str | Unset = UNSET,

) -> Response[Any | Error | File]:
    """  Session requise et règles de visibilité identiques au détail. Le chemin disque n’est jamais exposé.

    Args:
        id (int):
        range_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error | File]
     """


    kwargs = _get_kwargs(
        id=id,
range_=range_,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    range_: str | Unset = UNSET,

) -> Any | Error | File | None:
    """  Session requise et règles de visibilité identiques au détail. Le chemin disque n’est jamais exposé.

    Args:
        id (int):
        range_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error | File
     """


    return sync_detailed(
        id=id,
client=client,
range_=range_,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    range_: str | Unset = UNSET,

) -> Response[Any | Error | File]:
    """  Session requise et règles de visibilité identiques au détail. Le chemin disque n’est jamais exposé.

    Args:
        id (int):
        range_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error | File]
     """


    kwargs = _get_kwargs(
        id=id,
range_=range_,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    range_: str | Unset = UNSET,

) -> Any | Error | File | None:
    """  Session requise et règles de visibilité identiques au détail. Le chemin disque n’est jamais exposé.

    Args:
        id (int):
        range_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error | File
     """


    return (await asyncio_detailed(
        id=id,
client=client,
range_=range_,

    )).parsed
