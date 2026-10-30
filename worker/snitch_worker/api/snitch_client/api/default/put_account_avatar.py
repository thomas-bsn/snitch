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
    *,
    body:    File  |     File  | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/account/avatar",
    }

    if isinstance(body, File):
        _kwargs["content"] = body.payload
        headers["Content-Type"] = "image/jpeg"
    if isinstance(body, File):
        _kwargs["content"] = body.payload
        headers["Content-Type"] = "image/png"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Error:
    if response.status_code == 204:
        response_204 = cast(Any, None)
        return response_204

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body:    File  |     File  | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | Error]:
    """  Photo personnelle du compte connecté. JPEG/PNG, 5 Mo et 16 mégapixels maximum ; recadrage carré et
    stockage PNG 256×256 sans métadonnées.

    Args:
        origin (str):
        x_csrf_token (str):
        body (File):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    body:    File  |     File  | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Any | Error | None:
    """  Photo personnelle du compte connecté. JPEG/PNG, 5 Mo et 16 mégapixels maximum ; recadrage carré et
    stockage PNG 256×256 sans métadonnées.

    Args:
        origin (str):
        x_csrf_token (str):
        body (File):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body:    File  |     File  | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | Error]:
    """  Photo personnelle du compte connecté. JPEG/PNG, 5 Mo et 16 mégapixels maximum ; recadrage carré et
    stockage PNG 256×256 sans métadonnées.

    Args:
        origin (str):
        x_csrf_token (str):
        body (File):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body:    File  |     File  | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Any | Error | None:
    """  Photo personnelle du compte connecté. JPEG/PNG, 5 Mo et 16 mégapixels maximum ; recadrage carré et
    stockage PNG 256×256 sans métadonnées.

    Args:
        origin (str):
        x_csrf_token (str):
        body (File):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
