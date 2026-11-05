from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.put_worker_update_target_body import PutWorkerUpdateTargetBody
from ...models.put_worker_update_target_response_200 import PutWorkerUpdateTargetResponse200
from typing import cast



def _get_kwargs(
    *,
    body: PutWorkerUpdateTargetBody,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/admin/worker-updates",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | PutWorkerUpdateTargetResponse200:
    if response.status_code == 200:
        response_200 = PutWorkerUpdateTargetResponse200.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | PutWorkerUpdateTargetResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: PutWorkerUpdateTargetBody,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | PutWorkerUpdateTargetResponse200]:
    """  Choisit la stratégie : disabled (aucune mise à jour), pinned (version fixe, obligatoire) ou latest
    (suivi de la dernière version publiée, la version envoyée est ignorée). Sans mode, il est déduit de
    la version (vide : disabled). Le format est vérifié ; l'image est construite par Snitch à partir du
    dépôt configuré.

    Args:
        origin (str):
        x_csrf_token (str):
        body (PutWorkerUpdateTargetBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PutWorkerUpdateTargetResponse200]
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
    body: PutWorkerUpdateTargetBody,
    origin: str,
    x_csrf_token: str,

) -> Error | PutWorkerUpdateTargetResponse200 | None:
    """  Choisit la stratégie : disabled (aucune mise à jour), pinned (version fixe, obligatoire) ou latest
    (suivi de la dernière version publiée, la version envoyée est ignorée). Sans mode, il est déduit de
    la version (vide : disabled). Le format est vérifié ; l'image est construite par Snitch à partir du
    dépôt configuré.

    Args:
        origin (str):
        x_csrf_token (str):
        body (PutWorkerUpdateTargetBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PutWorkerUpdateTargetResponse200
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
    body: PutWorkerUpdateTargetBody,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | PutWorkerUpdateTargetResponse200]:
    """  Choisit la stratégie : disabled (aucune mise à jour), pinned (version fixe, obligatoire) ou latest
    (suivi de la dernière version publiée, la version envoyée est ignorée). Sans mode, il est déduit de
    la version (vide : disabled). Le format est vérifié ; l'image est construite par Snitch à partir du
    dépôt configuré.

    Args:
        origin (str):
        x_csrf_token (str):
        body (PutWorkerUpdateTargetBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PutWorkerUpdateTargetResponse200]
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
    body: PutWorkerUpdateTargetBody,
    origin: str,
    x_csrf_token: str,

) -> Error | PutWorkerUpdateTargetResponse200 | None:
    """  Choisit la stratégie : disabled (aucune mise à jour), pinned (version fixe, obligatoire) ou latest
    (suivi de la dernière version publiée, la version envoyée est ignorée). Sans mode, il est déduit de
    la version (vide : disabled). Le format est vérifié ; l'image est construite par Snitch à partir du
    dépôt configuré.

    Args:
        origin (str):
        x_csrf_token (str):
        body (PutWorkerUpdateTargetBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PutWorkerUpdateTargetResponse200
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
