from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_trace import VoiceTrace
from typing import cast



def _get_kwargs(
    id: int,

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/clips/{id}/voice-trace".format(id=quote(str(id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceTrace:
    if response.status_code == 200:
        response_200 = VoiceTrace.from_dict(response.json())



        return response_200

    if response.status_code == 404:
        response_404 = Error.from_dict(response.json())



        return response_404

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceTrace]:
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

) -> Response[Error | VoiceTrace]:
    """  Administrateurs uniquement. Étapes brutes autour de l'alignement entre la diarisation et la
    transcription (modèles et versions, tours, segments avec leurs voix candidates, voix retenue,
    couverture), sans aucun texte transcrit : joignable à un rapport sans exposer le clip. Diagnostic de
    #273.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceTrace]
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

) -> Error | VoiceTrace | None:
    """  Administrateurs uniquement. Étapes brutes autour de l'alignement entre la diarisation et la
    transcription (modèles et versions, tours, segments avec leurs voix candidates, voix retenue,
    couverture), sans aucun texte transcrit : joignable à un rapport sans exposer le clip. Diagnostic de
    #273.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceTrace
     """


    return sync_detailed(
        id=id,
client=client,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient,

) -> Response[Error | VoiceTrace]:
    """  Administrateurs uniquement. Étapes brutes autour de l'alignement entre la diarisation et la
    transcription (modèles et versions, tours, segments avec leurs voix candidates, voix retenue,
    couverture), sans aucun texte transcrit : joignable à un rapport sans exposer le clip. Diagnostic de
    #273.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceTrace]
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

) -> Error | VoiceTrace | None:
    """  Administrateurs uniquement. Étapes brutes autour de l'alignement entre la diarisation et la
    transcription (modèles et versions, tours, segments avec leurs voix candidates, voix retenue,
    couverture), sans aucun texte transcrit : joignable à un rapport sans exposer le clip. Diagnostic de
    #273.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceTrace
     """


    return (await asyncio_detailed(
        id=id,
client=client,

    )).parsed
