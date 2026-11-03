from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.worker_hf_token import WorkerHFToken
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/worker/hf-token",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | WorkerHFToken:
    if response.status_code == 200:
        response_200 = WorkerHFToken.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | WorkerHFToken]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | WorkerHFToken]:
    """  Token central pour un Worker authentifié et non révoqué, seulement lorsque sa version diffère de la
    sienne. Un token supprimé ou absent est renvoyé avec token null : le Worker purge alors sa copie
    centrale. Réponse sans cache.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerHFToken]
     """


    kwargs = _get_kwargs(

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,

) -> Error | WorkerHFToken | None:
    """  Token central pour un Worker authentifié et non révoqué, seulement lorsque sa version diffère de la
    sienne. Un token supprimé ou absent est renvoyé avec token null : le Worker purge alors sa copie
    centrale. Réponse sans cache.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerHFToken
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | WorkerHFToken]:
    """  Token central pour un Worker authentifié et non révoqué, seulement lorsque sa version diffère de la
    sienne. Un token supprimé ou absent est renvoyé avec token null : le Worker purge alors sa copie
    centrale. Réponse sans cache.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerHFToken]
     """


    kwargs = _get_kwargs(

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,

) -> Error | WorkerHFToken | None:
    """  Token central pour un Worker authentifié et non révoqué, seulement lorsque sa version diffère de la
    sienne. Un token supprimé ou absent est renvoyé avec token null : le Worker purge alors sa copie
    centrale. Réponse sans cache.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerHFToken
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
