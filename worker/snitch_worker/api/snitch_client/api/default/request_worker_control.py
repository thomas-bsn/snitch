from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.worker_control_input import WorkerControlInput
from ...models.worker_control_state import WorkerControlState
from typing import cast



def _get_kwargs(
    id: str,
    *,
    body: WorkerControlInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/workers/{id}/control".format(id=quote(str(id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | WorkerControlState:
    if response.status_code == 200:
        response_200 = WorkerControlState.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | WorkerControlState]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: str,
    *,
    client: AuthenticatedClient,
    body: WorkerControlInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | WorkerControlState]:
    """  Administrateurs uniquement. Met le Worker en pause (il ne prend plus de nouveau job ; ceux en cours
    se terminent) ou le reprend. Le Worker ne reçoit rien : il lit cet état à son prochain relevé. Le
    redémarrage du conteneur reste le redémarrage géré (`/api/admin/workers/{id}/restart`). La demande
    est tracée avec l'administrateur qui l'a faite.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (WorkerControlInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerControlState]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: str,
    *,
    client: AuthenticatedClient,
    body: WorkerControlInput,
    origin: str,
    x_csrf_token: str,

) -> Error | WorkerControlState | None:
    """  Administrateurs uniquement. Met le Worker en pause (il ne prend plus de nouveau job ; ceux en cours
    se terminent) ou le reprend. Le Worker ne reçoit rien : il lit cet état à son prochain relevé. Le
    redémarrage du conteneur reste le redémarrage géré (`/api/admin/workers/{id}/restart`). La demande
    est tracée avec l'administrateur qui l'a faite.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (WorkerControlInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerControlState
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: str,
    *,
    client: AuthenticatedClient,
    body: WorkerControlInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | WorkerControlState]:
    """  Administrateurs uniquement. Met le Worker en pause (il ne prend plus de nouveau job ; ceux en cours
    se terminent) ou le reprend. Le Worker ne reçoit rien : il lit cet état à son prochain relevé. Le
    redémarrage du conteneur reste le redémarrage géré (`/api/admin/workers/{id}/restart`). La demande
    est tracée avec l'administrateur qui l'a faite.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (WorkerControlInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerControlState]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: str,
    *,
    client: AuthenticatedClient,
    body: WorkerControlInput,
    origin: str,
    x_csrf_token: str,

) -> Error | WorkerControlState | None:
    """  Administrateurs uniquement. Met le Worker en pause (il ne prend plus de nouveau job ; ceux en cours
    se terminent) ou le reprend. Le Worker ne reçoit rien : il lit cet état à son prochain relevé. Le
    redémarrage du conteneur reste le redémarrage géré (`/api/admin/workers/{id}/restart`). La demande
    est tracée avec l'administrateur qui l'a faite.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (WorkerControlInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerControlState
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
