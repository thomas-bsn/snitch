from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_matching_status import VoiceMatchingStatus
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/voice-matching/status",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceMatchingStatus:
    if response.status_code == 200:
        response_200 = VoiceMatchingStatus.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceMatchingStatus]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | VoiceMatchingStatus]:
    """  Utilisateurs connectés. Après une validation, une correction ou un changement de réglage, le
    rapprochement suit en arrière-plan (#392) : cet état dit s'il est en cours et quand le dernier s'est
    terminé, donc si les suggestions affichées sont à jour.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceMatchingStatus]
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

) -> Error | VoiceMatchingStatus | None:
    """  Utilisateurs connectés. Après une validation, une correction ou un changement de réglage, le
    rapprochement suit en arrière-plan (#392) : cet état dit s'il est en cours et quand le dernier s'est
    terminé, donc si les suggestions affichées sont à jour.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceMatchingStatus
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | VoiceMatchingStatus]:
    """  Utilisateurs connectés. Après une validation, une correction ou un changement de réglage, le
    rapprochement suit en arrière-plan (#392) : cet état dit s'il est en cours et quand le dernier s'est
    terminé, donc si les suggestions affichées sont à jour.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceMatchingStatus]
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

) -> Error | VoiceMatchingStatus | None:
    """  Utilisateurs connectés. Après une validation, une correction ou un changement de réglage, le
    rapprochement suit en arrière-plan (#392) : cet état dit s'il est en cours et quand le dernier s'est
    terminé, donc si les suggestions affichées sont à jour.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceMatchingStatus
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
