from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.poll_worker_restart_response_200 import PollWorkerRestartResponse200
from typing import cast



def _get_kwargs(
    *,
    authorization: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Authorization"] = authorization







    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/worker/supervisor/restarts",
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | PollWorkerRestartResponse200:
    if response.status_code == 200:
        response_200 = PollWorkerRestartResponse200.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | PollWorkerRestartResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    authorization: str,

) -> Response[Error | PollWorkerRestartResponse200]:
    """  Polling authentifié du superviseur hôte.

    Args:
        authorization (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PollWorkerRestartResponse200]
     """


    kwargs = _get_kwargs(
        authorization=authorization,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    authorization: str,

) -> Error | PollWorkerRestartResponse200 | None:
    """  Polling authentifié du superviseur hôte.

    Args:
        authorization (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PollWorkerRestartResponse200
     """


    return sync_detailed(
        client=client,
authorization=authorization,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    authorization: str,

) -> Response[Error | PollWorkerRestartResponse200]:
    """  Polling authentifié du superviseur hôte.

    Args:
        authorization (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PollWorkerRestartResponse200]
     """


    kwargs = _get_kwargs(
        authorization=authorization,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    authorization: str,

) -> Error | PollWorkerRestartResponse200 | None:
    """  Polling authentifié du superviseur hôte.

    Args:
        authorization (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PollWorkerRestartResponse200
     """


    return (await asyncio_detailed(
        client=client,
authorization=authorization,

    )).parsed
