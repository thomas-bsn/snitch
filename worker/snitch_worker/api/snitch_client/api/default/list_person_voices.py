from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.person_voice_list import PersonVoiceList
from typing import cast



def _get_kwargs(
    id: int,

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/persons/{id}/voices".format(id=quote(str(id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | PersonVoiceList:
    if response.status_code == 200:
        response_200 = PersonVoiceList.from_dict(response.json())



        return response_200

    if response.status_code == 404:
        response_404 = Error.from_dict(response.json())



        return response_404

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | PersonVoiceList]:
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

) -> Response[Error | PersonVoiceList]:
    """  Utilisateurs connectés. Les voix qu'une séparation peut nommer (POST /api/persons/{id}/separation) :
    validation humaine, clip visible pour l'appelant. Extraits écoutables par GET /api/local-
    speakers/{id}/excerpt. Cette lecture n'écrit rien.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PersonVoiceList]
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

) -> Error | PersonVoiceList | None:
    """  Utilisateurs connectés. Les voix qu'une séparation peut nommer (POST /api/persons/{id}/separation) :
    validation humaine, clip visible pour l'appelant. Extraits écoutables par GET /api/local-
    speakers/{id}/excerpt. Cette lecture n'écrit rien.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PersonVoiceList
     """


    return sync_detailed(
        id=id,
client=client,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient,

) -> Response[Error | PersonVoiceList]:
    """  Utilisateurs connectés. Les voix qu'une séparation peut nommer (POST /api/persons/{id}/separation) :
    validation humaine, clip visible pour l'appelant. Extraits écoutables par GET /api/local-
    speakers/{id}/excerpt. Cette lecture n'écrit rien.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PersonVoiceList]
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

) -> Error | PersonVoiceList | None:
    """  Utilisateurs connectés. Les voix qu'une séparation peut nommer (POST /api/persons/{id}/separation) :
    validation humaine, clip visible pour l'appelant. Extraits écoutables par GET /api/local-
    speakers/{id}/excerpt. Cette lecture n'écrit rien.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PersonVoiceList
     """


    return (await asyncio_detailed(
        id=id,
client=client,

    )).parsed
