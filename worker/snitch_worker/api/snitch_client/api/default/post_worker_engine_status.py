from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.worker_engine_status import WorkerEngineStatus
from typing import cast



def _get_kwargs(
    *,
    body: WorkerEngineStatus,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}






    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/worker/voice-engines/status",
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
    body: WorkerEngineStatus,

) -> Response[Any | Error]:
    """  Le Worker rapporte l'état réel d'une version d'engine en réponse à un ordre. Un rapport tardif
    (génération plus ancienne, ordre annulé ou terminé, étape déjà dépassée) est ignoré et journalisé
    sans reculer l'état.

    Args:
        body (WorkerEngineStatus):

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
    body: WorkerEngineStatus,

) -> Any | Error | None:
    """  Le Worker rapporte l'état réel d'une version d'engine en réponse à un ordre. Un rapport tardif
    (génération plus ancienne, ordre annulé ou terminé, étape déjà dépassée) est ignoré et journalisé
    sans reculer l'état.

    Args:
        body (WorkerEngineStatus):

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
    body: WorkerEngineStatus,

) -> Response[Any | Error]:
    """  Le Worker rapporte l'état réel d'une version d'engine en réponse à un ordre. Un rapport tardif
    (génération plus ancienne, ordre annulé ou terminé, étape déjà dépassée) est ignoré et journalisé
    sans reculer l'état.

    Args:
        body (WorkerEngineStatus):

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
    body: WorkerEngineStatus,

) -> Any | Error | None:
    """  Le Worker rapporte l'état réel d'une version d'engine en réponse à un ordre. Un rapport tardif
    (génération plus ancienne, ordre annulé ou terminé, étape déjà dépassée) est ignoré et journalisé
    sans reculer l'état.

    Args:
        body (WorkerEngineStatus):

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
