from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...types import File, FileTypes
from io import BytesIO
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/account/avatar",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Error | File:
    if response.status_code == 200:
        response_200 = File(
             payload = BytesIO(response.content)
        )



        return response_200

    if response.status_code == 404:
        response_404 = cast(Any, None)
        return response_404

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
    *,
    client: AuthenticatedClient | Client,

) -> Response[Any | Error | File]:
    """  Photo personnelle du compte connecté. JPEG/PNG, 5 Mo et 16 mégapixels maximum ; recadrage carré et
    stockage PNG 256×256 sans métadonnées.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error | File]
     """


    kwargs = _get_kwargs(

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,

) -> Any | Error | File | None:
    """  Photo personnelle du compte connecté. JPEG/PNG, 5 Mo et 16 mégapixels maximum ; recadrage carré et
    stockage PNG 256×256 sans métadonnées.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error | File
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,

) -> Response[Any | Error | File]:
    """  Photo personnelle du compte connecté. JPEG/PNG, 5 Mo et 16 mégapixels maximum ; recadrage carré et
    stockage PNG 256×256 sans métadonnées.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error | File]
     """


    kwargs = _get_kwargs(

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,

) -> Any | Error | File | None:
    """  Photo personnelle du compte connecté. JPEG/PNG, 5 Mo et 16 mégapixels maximum ; recadrage carré et
    stockage PNG 256×256 sans métadonnées.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error | File
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
