from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.browse_video_root_response_200 import BrowseVideoRootResponse200
from ...models.error import Error
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    path: str | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["path"] = path


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/video-root/browse",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> BrowseVideoRootResponse200 | Error:
    if response.status_code == 200:
        response_200 = BrowseVideoRootResponse200.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[BrowseVideoRootResponse200 | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    path: str | Unset = UNSET,

) -> Response[BrowseVideoRootResponse200 | Error]:
    """
    Args:
        path (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BrowseVideoRootResponse200 | Error]
     """


    kwargs = _get_kwargs(
        path=path,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    path: str | Unset = UNSET,

) -> BrowseVideoRootResponse200 | Error | None:
    """
    Args:
        path (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BrowseVideoRootResponse200 | Error
     """


    return sync_detailed(
        client=client,
path=path,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    path: str | Unset = UNSET,

) -> Response[BrowseVideoRootResponse200 | Error]:
    """
    Args:
        path (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BrowseVideoRootResponse200 | Error]
     """


    kwargs = _get_kwargs(
        path=path,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    path: str | Unset = UNSET,

) -> BrowseVideoRootResponse200 | Error | None:
    """
    Args:
        path (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BrowseVideoRootResponse200 | Error
     """


    return (await asyncio_detailed(
        client=client,
path=path,

    )).parsed
