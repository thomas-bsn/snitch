from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.runtime_bindings import RuntimeBindings
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/voice-engines/bindings",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | RuntimeBindings:
    if response.status_code == 200:
        response_200 = RuntimeBindings.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | RuntimeBindings]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,

) -> Response[Error | RuntimeBindings]:
    """  Administrateur : pour chaque modèle prêt sur un Worker et fondé sur un engine approuvé, la version
    liée, la version précédente (retour arrière), l'ordre en cours et l'historique des mises à jour.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | RuntimeBindings]
     """


    kwargs = _get_kwargs(

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,

) -> Error | RuntimeBindings | None:
    """  Administrateur : pour chaque modèle prêt sur un Worker et fondé sur un engine approuvé, la version
    liée, la version précédente (retour arrière), l'ordre en cours et l'historique des mises à jour.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | RuntimeBindings
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,

) -> Response[Error | RuntimeBindings]:
    """  Administrateur : pour chaque modèle prêt sur un Worker et fondé sur un engine approuvé, la version
    liée, la version précédente (retour arrière), l'ordre en cours et l'historique des mises à jour.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | RuntimeBindings]
     """


    kwargs = _get_kwargs(

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,

) -> Error | RuntimeBindings | None:
    """  Administrateur : pour chaque modèle prêt sur un Worker et fondé sur un engine approuvé, la version
    liée, la version précédente (retour arrière), l'ordre en cours et l'historique des mises à jour.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | RuntimeBindings
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
