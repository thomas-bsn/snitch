from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.reprocess_clip_metadata_response_200 import ReprocessClipMetadataResponse200
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    after: int | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token





    params: dict[str, Any] = {}

    params["after"] = after


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/clip-metadata/reprocess",
        "params": params,
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | ReprocessClipMetadataResponse200:
    if response.status_code == 200:
        response_200 = ReprocessClipMetadataResponse200.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | ReprocessClipMetadataResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    after: int | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | ReprocessClipMetadataResponse200]:
    """
    Args:
        after (int | Unset):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ReprocessClipMetadataResponse200]
     """


    kwargs = _get_kwargs(
        after=after,
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
    after: int | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Error | ReprocessClipMetadataResponse200 | None:
    """
    Args:
        after (int | Unset):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ReprocessClipMetadataResponse200
     """


    return sync_detailed(
        client=client,
after=after,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    after: int | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | ReprocessClipMetadataResponse200]:
    """
    Args:
        after (int | Unset):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ReprocessClipMetadataResponse200]
     """


    kwargs = _get_kwargs(
        after=after,
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
    after: int | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Error | ReprocessClipMetadataResponse200 | None:
    """
    Args:
        after (int | Unset):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ReprocessClipMetadataResponse200
     """


    return (await asyncio_detailed(
        client=client,
after=after,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
