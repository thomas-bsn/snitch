from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_models import VoiceModels
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/voice-models",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceModels:
    if response.status_code == 200:
        response_200 = VoiceModels.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceModels]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | VoiceModels]:
    """  Administrateurs uniquement. Catalogue des modèles vocaux (données figées dans le code : moteur,
    dépôt, révision épinglée, licence, accès gated, limites) avec, pour chacun, l'état réel rapporté par
    chaque Worker. Un modèle est installé si au moins un Worker l'a prêt.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModels]
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

) -> Error | VoiceModels | None:
    """  Administrateurs uniquement. Catalogue des modèles vocaux (données figées dans le code : moteur,
    dépôt, révision épinglée, licence, accès gated, limites) avec, pour chacun, l'état réel rapporté par
    chaque Worker. Un modèle est installé si au moins un Worker l'a prêt.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModels
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | VoiceModels]:
    """  Administrateurs uniquement. Catalogue des modèles vocaux (données figées dans le code : moteur,
    dépôt, révision épinglée, licence, accès gated, limites) avec, pour chacun, l'état réel rapporté par
    chaque Worker. Un modèle est installé si au moins un Worker l'a prêt.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModels]
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

) -> Error | VoiceModels | None:
    """  Administrateurs uniquement. Catalogue des modèles vocaux (données figées dans le code : moteur,
    dépôt, révision épinglée, licence, accès gated, limites) avec, pour chacun, l'état réel rapporté par
    chaque Worker. Un modèle est installé si au moins un Worker l'a prêt.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModels
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
