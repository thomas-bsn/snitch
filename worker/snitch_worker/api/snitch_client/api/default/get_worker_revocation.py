from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.get_worker_revocation_response_200 import GetWorkerRevocationResponse200
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/worker/revocation",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | GetWorkerRevocationResponse200:
    if response.status_code == 200:
        response_200 = GetWorkerRevocationResponse200.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | GetWorkerRevocationResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | GetWorkerRevocationResponse200]:
    """  Confirmation étroite pour le lanceur local, seul endroit où un jeton révoqué est accepté. La réponse
    dit seulement si l'identité est révoquée et depuis quand : aucune configuration, aucune action. Une
    réponse révoquée est enregistrée comme confirmation du lanceur, qui arrête alors le service. Un
    jeton inconnu reçoit worker_token_invalid.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetWorkerRevocationResponse200]
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

) -> Error | GetWorkerRevocationResponse200 | None:
    """  Confirmation étroite pour le lanceur local, seul endroit où un jeton révoqué est accepté. La réponse
    dit seulement si l'identité est révoquée et depuis quand : aucune configuration, aucune action. Une
    réponse révoquée est enregistrée comme confirmation du lanceur, qui arrête alors le service. Un
    jeton inconnu reçoit worker_token_invalid.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetWorkerRevocationResponse200
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | GetWorkerRevocationResponse200]:
    """  Confirmation étroite pour le lanceur local, seul endroit où un jeton révoqué est accepté. La réponse
    dit seulement si l'identité est révoquée et depuis quand : aucune configuration, aucune action. Une
    réponse révoquée est enregistrée comme confirmation du lanceur, qui arrête alors le service. Un
    jeton inconnu reçoit worker_token_invalid.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetWorkerRevocationResponse200]
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

) -> Error | GetWorkerRevocationResponse200 | None:
    """  Confirmation étroite pour le lanceur local, seul endroit où un jeton révoqué est accepté. La réponse
    dit seulement si l'identité est révoquée et depuis quand : aucune configuration, aucune action. Une
    réponse révoquée est enregistrée comme confirmation du lanceur, qui arrête alors le service. Un
    jeton inconnu reçoit worker_token_invalid.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetWorkerRevocationResponse200
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
