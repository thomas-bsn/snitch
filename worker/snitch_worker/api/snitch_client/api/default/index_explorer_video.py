from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.explorer_index_result import ExplorerIndexResult
from ...models.index_explorer_video_body import IndexExplorerVideoBody
from typing import cast



def _get_kwargs(
    *,
    body: IndexExplorerVideoBody,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}






    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/library/explorer/index",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | ExplorerIndexResult:
    if response.status_code == 200:
        response_200 = ExplorerIndexResult.from_dict(response.json())



        return response_200

    if response.status_code == 202:
        response_202 = ExplorerIndexResult.from_dict(response.json())



        return response_202

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | ExplorerIndexResult]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: IndexExplorerVideoBody,

) -> Response[Error | ExplorerIndexResult]:
    """ Demander l'indexation d'une vidéo ouverte dans l'explorateur, sans analyse

    Args:
        body (IndexExplorerVideoBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ExplorerIndexResult]
     """


    kwargs = _get_kwargs(
        body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    body: IndexExplorerVideoBody,

) -> Error | ExplorerIndexResult | None:
    """ Demander l'indexation d'une vidéo ouverte dans l'explorateur, sans analyse

    Args:
        body (IndexExplorerVideoBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ExplorerIndexResult
     """


    return sync_detailed(
        client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: IndexExplorerVideoBody,

) -> Response[Error | ExplorerIndexResult]:
    """ Demander l'indexation d'une vidéo ouverte dans l'explorateur, sans analyse

    Args:
        body (IndexExplorerVideoBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ExplorerIndexResult]
     """


    kwargs = _get_kwargs(
        body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: IndexExplorerVideoBody,

) -> Error | ExplorerIndexResult | None:
    """ Demander l'indexation d'une vidéo ouverte dans l'explorateur, sans analyse

    Args:
        body (IndexExplorerVideoBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ExplorerIndexResult
     """


    return (await asyncio_detailed(
        client=client,
body=body,

    )).parsed
