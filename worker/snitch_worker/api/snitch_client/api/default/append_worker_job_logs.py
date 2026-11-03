from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.append_worker_job_logs_response_200 import AppendWorkerJobLogsResponse200
from ...models.error import Error
from ...models.worker_job_logs_input import WorkerJobLogsInput
from typing import cast



def _get_kwargs(
    id: int,
    *,
    body: WorkerJobLogsInput,
    x_worker_lease_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["X-Worker-Lease-Token"] = x_worker_lease_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/worker/jobs/{id}/logs".format(id=quote(str(id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AppendWorkerJobLogsResponse200 | Error:
    if response.status_code == 200:
        response_200 = AppendWorkerJobLogsResponse200.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AppendWorkerJobLogsResponse200 | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    body: WorkerJobLogsInput,
    x_worker_lease_token: str,

) -> Response[AppendWorkerJobLogsResponse200 | Error]:
    """  Lignes de log d'un job envoyées par le Worker par lots (2 s ou 50 lignes), déjà masquées côté
    Worker. Le serveur ne garde que les 50 dernières lignes du job ; elles sont supprimées avec le job.
    Lease obligatoire.

    Args:
        id (int):
        x_worker_lease_token (str):
        body (WorkerJobLogsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AppendWorkerJobLogsResponse200 | Error]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
x_worker_lease_token=x_worker_lease_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    *,
    client: AuthenticatedClient,
    body: WorkerJobLogsInput,
    x_worker_lease_token: str,

) -> AppendWorkerJobLogsResponse200 | Error | None:
    """  Lignes de log d'un job envoyées par le Worker par lots (2 s ou 50 lignes), déjà masquées côté
    Worker. Le serveur ne garde que les 50 dernières lignes du job ; elles sont supprimées avec le job.
    Lease obligatoire.

    Args:
        id (int):
        x_worker_lease_token (str):
        body (WorkerJobLogsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AppendWorkerJobLogsResponse200 | Error
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
x_worker_lease_token=x_worker_lease_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    body: WorkerJobLogsInput,
    x_worker_lease_token: str,

) -> Response[AppendWorkerJobLogsResponse200 | Error]:
    """  Lignes de log d'un job envoyées par le Worker par lots (2 s ou 50 lignes), déjà masquées côté
    Worker. Le serveur ne garde que les 50 dernières lignes du job ; elles sont supprimées avec le job.
    Lease obligatoire.

    Args:
        id (int):
        x_worker_lease_token (str):
        body (WorkerJobLogsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AppendWorkerJobLogsResponse200 | Error]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
x_worker_lease_token=x_worker_lease_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    *,
    client: AuthenticatedClient,
    body: WorkerJobLogsInput,
    x_worker_lease_token: str,

) -> AppendWorkerJobLogsResponse200 | Error | None:
    """  Lignes de log d'un job envoyées par le Worker par lots (2 s ou 50 lignes), déjà masquées côté
    Worker. Le serveur ne garde que les 50 dernières lignes du job ; elles sont supprimées avec le job.
    Lease obligatoire.

    Args:
        id (int):
        x_worker_lease_token (str):
        body (WorkerJobLogsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AppendWorkerJobLogsResponse200 | Error
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
x_worker_lease_token=x_worker_lease_token,

    )).parsed
