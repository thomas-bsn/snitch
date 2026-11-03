from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_refusal import VoiceRefusal
from typing import cast



def _get_kwargs(
    id: int,
    person_id: int,
    *,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/local-speakers/{id}/refusals/{person_id}".format(id=quote(str(id), safe=""),person_id=quote(str(person_id), safe=""),),
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceRefusal:
    if response.status_code == 200:
        response_200 = VoiceRefusal.from_dict(response.json())



        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())



        return response_400

    if response.status_code == 409:
        response_409 = Error.from_dict(response.json())



        return response_409

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceRefusal]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    person_id: int,
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceRefusal]:
    """  Utilisateurs connectés, pour un clip visible : même droit que le refus (ADR 0031). « Annuler le
    refus » : la Personne n'est plus exclue du rapprochement pour cette voix. L'attribution retirée par
    le refus n'est pas rétablie.

    Args:
        id (int):
        person_id (int):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceRefusal]
     """


    kwargs = _get_kwargs(
        id=id,
person_id=person_id,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    person_id: int,
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceRefusal | None:
    """  Utilisateurs connectés, pour un clip visible : même droit que le refus (ADR 0031). « Annuler le
    refus » : la Personne n'est plus exclue du rapprochement pour cette voix. L'attribution retirée par
    le refus n'est pas rétablie.

    Args:
        id (int):
        person_id (int):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceRefusal
     """


    return sync_detailed(
        id=id,
person_id=person_id,
client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    person_id: int,
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceRefusal]:
    """  Utilisateurs connectés, pour un clip visible : même droit que le refus (ADR 0031). « Annuler le
    refus » : la Personne n'est plus exclue du rapprochement pour cette voix. L'attribution retirée par
    le refus n'est pas rétablie.

    Args:
        id (int):
        person_id (int):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceRefusal]
     """


    kwargs = _get_kwargs(
        id=id,
person_id=person_id,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    person_id: int,
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceRefusal | None:
    """  Utilisateurs connectés, pour un clip visible : même droit que le refus (ADR 0031). « Annuler le
    refus » : la Personne n'est plus exclue du rapprochement pour cette voix. L'attribution retirée par
    le refus n'est pas rétablie.

    Args:
        id (int):
        person_id (int):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceRefusal
     """


    return (await asyncio_detailed(
        id=id,
person_id=person_id,
client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
