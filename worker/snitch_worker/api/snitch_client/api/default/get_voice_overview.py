from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_overview import VoiceOverview
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/voice-overview",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceOverview:
    if response.status_code == 200:
        response_200 = VoiceOverview.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceOverview]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | VoiceOverview]:
    """  Administrateurs uniquement. Les modèles vocaux que chaque Worker déclare réellement (jamais une
    saisie), leur état et le nombre de travaux vocaux en attente d'un Worker prêt. Un Worker n'est prêt
    que s'il détient les modèles actifs et qu'une inférence de test a réussi.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceOverview]
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

) -> Error | VoiceOverview | None:
    """  Administrateurs uniquement. Les modèles vocaux que chaque Worker déclare réellement (jamais une
    saisie), leur état et le nombre de travaux vocaux en attente d'un Worker prêt. Un Worker n'est prêt
    que s'il détient les modèles actifs et qu'une inférence de test a réussi.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceOverview
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | VoiceOverview]:
    """  Administrateurs uniquement. Les modèles vocaux que chaque Worker déclare réellement (jamais une
    saisie), leur état et le nombre de travaux vocaux en attente d'un Worker prêt. Un Worker n'est prêt
    que s'il détient les modèles actifs et qu'une inférence de test a réussi.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceOverview]
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

) -> Error | VoiceOverview | None:
    """  Administrateurs uniquement. Les modèles vocaux que chaque Worker déclare réellement (jamais une
    saisie), leur état et le nombre de travaux vocaux en attente d'un Worker prêt. Un Worker n'est prêt
    que s'il détient les modèles actifs et qu'une inférence de test a réussi.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceOverview
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
