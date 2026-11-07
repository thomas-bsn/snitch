from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.update_worker_restart_body import UpdateWorkerRestartBody
from typing import cast



def _get_kwargs(
    id: str,
    *,
    body: UpdateWorkerRestartBody,
    authorization: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Authorization"] = authorization







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/worker/supervisor/restarts/{id}".format(id=quote(str(id), safe=""),),
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
    id: str,
    *,
    client: AuthenticatedClient | Client,
    body: UpdateWorkerRestartBody,
    authorization: str,

) -> Response[Any | Error]:
    """  Mise à jour authentifiée de l’état de commande par le superviseur hôte.

    Args:
        id (str):
        authorization (str):
        body (UpdateWorkerRestartBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
authorization=authorization,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    body: UpdateWorkerRestartBody,
    authorization: str,

) -> Any | Error | None:
    """  Mise à jour authentifiée de l’état de commande par le superviseur hôte.

    Args:
        id (str):
        authorization (str):
        body (UpdateWorkerRestartBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
authorization=authorization,

    ).parsed

async def asyncio_detailed(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    body: UpdateWorkerRestartBody,
    authorization: str,

) -> Response[Any | Error]:
    """  Mise à jour authentifiée de l’état de commande par le superviseur hôte.

    Args:
        id (str):
        authorization (str):
        body (UpdateWorkerRestartBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
authorization=authorization,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    body: UpdateWorkerRestartBody,
    authorization: str,

) -> Any | Error | None:
    """  Mise à jour authentifiée de l’état de commande par le superviseur hôte.

    Args:
        id (str):
        authorization (str):
        body (UpdateWorkerRestartBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
authorization=authorization,

    )).parsed
