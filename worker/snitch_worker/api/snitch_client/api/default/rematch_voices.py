from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_matching_summary import VoiceMatchingSummary
from typing import cast



def _get_kwargs(
    *,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/voice-matching/rematch",
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceMatchingSummary:
    if response.status_code == 200:
        response_200 = VoiceMatchingSummary.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceMatchingSummary]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceMatchingSummary]:
    """  Administrateurs uniquement. Recalcule les attributions automatiques à partir des empreintes et des
    références actuelles ; ne crée jamais d'Échantillon vocal validé. Exécute tout de suite un passage
    complet, qui prend aussi en compte les demandes en attente, et en renvoie les chiffres.

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceMatchingSummary]
     """


    kwargs = _get_kwargs(
        origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceMatchingSummary | None:
    """  Administrateurs uniquement. Recalcule les attributions automatiques à partir des empreintes et des
    références actuelles ; ne crée jamais d'Échantillon vocal validé. Exécute tout de suite un passage
    complet, qui prend aussi en compte les demandes en attente, et en renvoie les chiffres.

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceMatchingSummary
     """


    return sync_detailed(
        client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceMatchingSummary]:
    """  Administrateurs uniquement. Recalcule les attributions automatiques à partir des empreintes et des
    références actuelles ; ne crée jamais d'Échantillon vocal validé. Exécute tout de suite un passage
    complet, qui prend aussi en compte les demandes en attente, et en renvoie les chiffres.

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceMatchingSummary]
     """


    kwargs = _get_kwargs(
        origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceMatchingSummary | None:
    """  Administrateurs uniquement. Recalcule les attributions automatiques à partir des empreintes et des
    références actuelles ; ne crée jamais d'Échantillon vocal validé. Exécute tout de suite un passage
    complet, qui prend aussi en compte les demandes en attente, et en renvoie les chiffres.

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceMatchingSummary
     """


    return (await asyncio_detailed(
        client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
