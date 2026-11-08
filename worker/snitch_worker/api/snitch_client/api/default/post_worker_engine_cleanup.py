from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.engine_cleanup_report import EngineCleanupReport
from ...models.error import Error
from typing import cast



def _get_kwargs(
    *,
    body: EngineCleanupReport,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}






    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/worker/voice-engines/cleanup",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

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
    client: AuthenticatedClient,
    body: EngineCleanupReport,

) -> Response[Any | Error]:
    """  Le Worker confirme qu'il a exécuté un nettoyage demandé (fichiers supprimés ou rien à supprimer).
    Rejouer la confirmation est sans effet ; une confirmation non demandée est refusée.

    Args:
        body (EngineCleanupReport):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
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
    client: AuthenticatedClient,
    body: EngineCleanupReport,

) -> Any | Error | None:
    """  Le Worker confirme qu'il a exécuté un nettoyage demandé (fichiers supprimés ou rien à supprimer).
    Rejouer la confirmation est sans effet ; une confirmation non demandée est refusée.

    Args:
        body (EngineCleanupReport):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return sync_detailed(
        client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: EngineCleanupReport,

) -> Response[Any | Error]:
    """  Le Worker confirme qu'il a exécuté un nettoyage demandé (fichiers supprimés ou rien à supprimer).
    Rejouer la confirmation est sans effet ; une confirmation non demandée est refusée.

    Args:
        body (EngineCleanupReport):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
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
    client: AuthenticatedClient,
    body: EngineCleanupReport,

) -> Any | Error | None:
    """  Le Worker confirme qu'il a exécuté un nettoyage demandé (fichiers supprimés ou rien à supprimer).
    Rejouer la confirmation est sans effet ; une confirmation non demandée est refusée.

    Args:
        body (EngineCleanupReport):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return (await asyncio_detailed(
        client=client,
body=body,

    )).parsed
