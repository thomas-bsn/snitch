from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.person_list import PersonList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    q: str | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["q"] = q


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/persons",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | PersonList:
    if response.status_code == 200:
        response_200 = PersonList.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | PersonList]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,

) -> Response[Error | PersonList]:
    """  Utilisateurs connectés. Chaque Personne porte le nombre de ses Échantillons vocaux validés actifs
    (références).

    Args:
        q (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PersonList]
     """


    kwargs = _get_kwargs(
        q=q,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,

) -> Error | PersonList | None:
    """  Utilisateurs connectés. Chaque Personne porte le nombre de ses Échantillons vocaux validés actifs
    (références).

    Args:
        q (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PersonList
     """


    return sync_detailed(
        client=client,
q=q,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,

) -> Response[Error | PersonList]:
    """  Utilisateurs connectés. Chaque Personne porte le nombre de ses Échantillons vocaux validés actifs
    (références).

    Args:
        q (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PersonList]
     """


    kwargs = _get_kwargs(
        q=q,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,

) -> Error | PersonList | None:
    """  Utilisateurs connectés. Chaque Personne porte le nombre de ses Échantillons vocaux validés actifs
    (références).

    Args:
        q (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PersonList
     """


    return (await asyncio_detailed(
        client=client,
q=q,

    )).parsed
