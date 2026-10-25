from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.attribution_history import AttributionHistory
from ...models.error import Error
from typing import cast



def _get_kwargs(
    id: int,

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/local-speakers/{id}/history".format(id=quote(str(id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AttributionHistory | Error:
    if response.status_code == 200:
        response_200 = AttributionHistory.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AttributionHistory | Error]:
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

) -> Response[AttributionHistory | Error]:
    """  Utilisateurs connectés, pour un clip visible. Ancienne valeur, nouvelle valeur, auteur et date de
    chaque validation ou correction ; aucune donnée vocale.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AttributionHistory | Error]
     """


    kwargs = _get_kwargs(
        id=id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    *,
    client: AuthenticatedClient,

) -> AttributionHistory | Error | None:
    """  Utilisateurs connectés, pour un clip visible. Ancienne valeur, nouvelle valeur, auteur et date de
    chaque validation ou correction ; aucune donnée vocale.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AttributionHistory | Error
     """


    return sync_detailed(
        id=id,
client=client,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient,

) -> Response[AttributionHistory | Error]:
    """  Utilisateurs connectés, pour un clip visible. Ancienne valeur, nouvelle valeur, auteur et date de
    chaque validation ou correction ; aucune donnée vocale.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AttributionHistory | Error]
     """


    kwargs = _get_kwargs(
        id=id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    *,
    client: AuthenticatedClient,

) -> AttributionHistory | Error | None:
    """  Utilisateurs connectés, pour un clip visible. Ancienne valeur, nouvelle valeur, auteur et date de
    chaque validation ou correction ; aucune donnée vocale.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AttributionHistory | Error
     """


    return (await asyncio_detailed(
        id=id,
client=client,

    )).parsed
