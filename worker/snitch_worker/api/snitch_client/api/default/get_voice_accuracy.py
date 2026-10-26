from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_accuracy import VoiceAccuracy
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/voice-accuracy",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceAccuracy:
    if response.status_code == 200:
        response_200 = VoiceAccuracy.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceAccuracy]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | VoiceAccuracy]:
    """  Administrateurs uniquement. Précision globale et par Personne mesurée sur des clips distincts de
    ceux des références (un échantillon validé est référence ou test, jamais les deux, et aucun test ne
    partage un clip avec une référence), nombre de confirmations, couverture automatique et volume de
    vérification restant, calculés depuis la base. Ces valeurs décrivent l'échantillon disponible :
    aucune n'est une garantie.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceAccuracy]
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

) -> Error | VoiceAccuracy | None:
    """  Administrateurs uniquement. Précision globale et par Personne mesurée sur des clips distincts de
    ceux des références (un échantillon validé est référence ou test, jamais les deux, et aucun test ne
    partage un clip avec une référence), nombre de confirmations, couverture automatique et volume de
    vérification restant, calculés depuis la base. Ces valeurs décrivent l'échantillon disponible :
    aucune n'est une garantie.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceAccuracy
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | VoiceAccuracy]:
    """  Administrateurs uniquement. Précision globale et par Personne mesurée sur des clips distincts de
    ceux des références (un échantillon validé est référence ou test, jamais les deux, et aucun test ne
    partage un clip avec une référence), nombre de confirmations, couverture automatique et volume de
    vérification restant, calculés depuis la base. Ces valeurs décrivent l'échantillon disponible :
    aucune n'est une garantie.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceAccuracy]
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

) -> Error | VoiceAccuracy | None:
    """  Administrateurs uniquement. Précision globale et par Personne mesurée sur des clips distincts de
    ceux des références (un échantillon validé est référence ou test, jamais les deux, et aucun test ne
    partage un clip avec une référence), nombre de confirmations, couverture automatique et volume de
    vérification restant, calculés depuis la base. Ces valeurs décrivent l'échantillon disponible :
    aucune n'est une garantie.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceAccuracy
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
