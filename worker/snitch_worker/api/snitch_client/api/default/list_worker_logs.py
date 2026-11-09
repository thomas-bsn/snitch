from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.worker_log_page import WorkerLogPage
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    id: str,
    *,
    after: int | Unset = UNSET,
    before: int | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["after"] = after

    params["before"] = before

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/workers/{id}/logs".format(id=quote(str(id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | WorkerLogPage:
    if response.status_code == 200:
        response_200 = WorkerLogPage.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | WorkerLogPage]:
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
    after: int | Unset = UNSET,
    before: int | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Response[Error | WorkerLogPage]:
    """  Administrateurs uniquement. Journal de la session en cours du Worker (son dernier processus), depuis
    son démarrage. Sans curseur : la page la plus récente. `after` suit la fin du journal en direct,
    `before` remonte vers le début de la session.

    Args:
        id (str):
        after (int | Unset):
        before (int | Unset):
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerLogPage]
     """


    kwargs = _get_kwargs(
        id=id,
after=after,
before=before,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: str,
    *,
    client: AuthenticatedClient,
    after: int | Unset = UNSET,
    before: int | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Error | WorkerLogPage | None:
    """  Administrateurs uniquement. Journal de la session en cours du Worker (son dernier processus), depuis
    son démarrage. Sans curseur : la page la plus récente. `after` suit la fin du journal en direct,
    `before` remonte vers le début de la session.

    Args:
        id (str):
        after (int | Unset):
        before (int | Unset):
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerLogPage
     """


    return sync_detailed(
        id=id,
client=client,
after=after,
before=before,
limit=limit,

    ).parsed

async def asyncio_detailed(
    id: str,
    *,
    client: AuthenticatedClient,
    after: int | Unset = UNSET,
    before: int | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Response[Error | WorkerLogPage]:
    """  Administrateurs uniquement. Journal de la session en cours du Worker (son dernier processus), depuis
    son démarrage. Sans curseur : la page la plus récente. `after` suit la fin du journal en direct,
    `before` remonte vers le début de la session.

    Args:
        id (str):
        after (int | Unset):
        before (int | Unset):
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerLogPage]
     """


    kwargs = _get_kwargs(
        id=id,
after=after,
before=before,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: str,
    *,
    client: AuthenticatedClient,
    after: int | Unset = UNSET,
    before: int | Unset = UNSET,
    limit: int | Unset = UNSET,

) -> Error | WorkerLogPage | None:
    """  Administrateurs uniquement. Journal de la session en cours du Worker (son dernier processus), depuis
    son démarrage. Sans curseur : la page la plus récente. `after` suit la fin du journal en direct,
    `before` remonte vers le début de la session.

    Args:
        id (str):
        after (int | Unset):
        before (int | Unset):
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerLogPage
     """


    return (await asyncio_detailed(
        id=id,
client=client,
after=after,
before=before,
limit=limit,

    )).parsed
