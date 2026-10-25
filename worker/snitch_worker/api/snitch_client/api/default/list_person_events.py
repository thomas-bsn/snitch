from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.person_event_list import PersonEventList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    person_id: int | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["person_id"] = person_id


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/person-events",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | PersonEventList:
    if response.status_code == 200:
        response_200 = PersonEventList.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | PersonEventList]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    person_id: int | Unset = UNSET,

) -> Response[Error | PersonEventList]:
    """  Utilisateurs connectés. Trace immuable (auteur, date, noms au moment du changement, aucun contenu
    vocal), y compris pour une Personne qui n'existe plus ; person_id retient aussi les fusions dont
    elle est la cible.

    Args:
        person_id (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PersonEventList]
     """


    kwargs = _get_kwargs(
        person_id=person_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    person_id: int | Unset = UNSET,

) -> Error | PersonEventList | None:
    """  Utilisateurs connectés. Trace immuable (auteur, date, noms au moment du changement, aucun contenu
    vocal), y compris pour une Personne qui n'existe plus ; person_id retient aussi les fusions dont
    elle est la cible.

    Args:
        person_id (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PersonEventList
     """


    return sync_detailed(
        client=client,
person_id=person_id,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    person_id: int | Unset = UNSET,

) -> Response[Error | PersonEventList]:
    """  Utilisateurs connectés. Trace immuable (auteur, date, noms au moment du changement, aucun contenu
    vocal), y compris pour une Personne qui n'existe plus ; person_id retient aussi les fusions dont
    elle est la cible.

    Args:
        person_id (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PersonEventList]
     """


    kwargs = _get_kwargs(
        person_id=person_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    person_id: int | Unset = UNSET,

) -> Error | PersonEventList | None:
    """  Utilisateurs connectés. Trace immuable (auteur, date, noms au moment du changement, aucun contenu
    vocal), y compris pour une Personne qui n'existe plus ; person_id retient aussi les fusions dont
    elle est la cible.

    Args:
        person_id (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PersonEventList
     """


    return (await asyncio_detailed(
        client=client,
person_id=person_id,

    )).parsed
