from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.revoke_worker_body import RevokeWorkerBody
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    id: str,
    *,
    body: RevokeWorkerBody | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/admin/workers/{id}".format(id=quote(str(id), safe=""),),
    }


    if not isinstance(body, Unset):
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
    body: RevokeWorkerBody | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | Error]:
    """  Administrateurs uniquement. Supprime le Worker : son jeton est révoqué immédiatement et pour
    toujours, ses tâches en cours reviennent aussitôt dans la file, ses résultats tardifs sont refusés,
    et seul l'audit minimal (nom, date, motif facultatif) est conservé. Le disque local n'est pas
    effacé. Une seconde suppression est sans effet et garde le premier audit.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (RevokeWorkerBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
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
    client: AuthenticatedClient | Client,
    body: RevokeWorkerBody | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Any | Error | None:
    """  Administrateurs uniquement. Supprime le Worker : son jeton est révoqué immédiatement et pour
    toujours, ses tâches en cours reviennent aussitôt dans la file, ses résultats tardifs sont refusés,
    et seul l'audit minimal (nom, date, motif facultatif) est conservé. Le disque local n'est pas
    effacé. Une seconde suppression est sans effet et garde le premier audit.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (RevokeWorkerBody | Unset):

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
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    body: RevokeWorkerBody | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | Error]:
    """  Administrateurs uniquement. Supprime le Worker : son jeton est révoqué immédiatement et pour
    toujours, ses tâches en cours reviennent aussitôt dans la file, ses résultats tardifs sont refusés,
    et seul l'audit minimal (nom, date, motif facultatif) est conservé. Le disque local n'est pas
    effacé. Une seconde suppression est sans effet et garde le premier audit.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (RevokeWorkerBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
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
    client: AuthenticatedClient | Client,
    body: RevokeWorkerBody | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Any | Error | None:
    """  Administrateurs uniquement. Supprime le Worker : son jeton est révoqué immédiatement et pour
    toujours, ses tâches en cours reviennent aussitôt dans la file, ses résultats tardifs sont refusés,
    et seul l'audit minimal (nom, date, motif facultatif) est conservé. Le disque local n'est pas
    effacé. Une seconde suppression est sans effet et garde le premier audit.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (RevokeWorkerBody | Unset):

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
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
