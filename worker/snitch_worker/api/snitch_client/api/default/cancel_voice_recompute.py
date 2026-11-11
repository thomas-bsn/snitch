from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_recompute_state import VoiceRecomputeState
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
        "method": "delete",
        "url": "/api/admin/voice-fingerprint-recompute",
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceRecomputeState:
    if response.status_code == 200:
        response_200 = VoiceRecomputeState.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceRecomputeState]:
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

) -> Response[Error | VoiceRecomputeState]:
    """  Administrateurs uniquement. Annule le recalcul en cours : seules ses propres tâches sont annulées,
    les empreintes déjà recalculées restent, le rapport est figé et la bibliothèque rapprochée de
    nouveau (voice_recompute_not_running sans recalcul en cours).

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceRecomputeState]
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

) -> Error | VoiceRecomputeState | None:
    """  Administrateurs uniquement. Annule le recalcul en cours : seules ses propres tâches sont annulées,
    les empreintes déjà recalculées restent, le rapport est figé et la bibliothèque rapprochée de
    nouveau (voice_recompute_not_running sans recalcul en cours).

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceRecomputeState
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

) -> Response[Error | VoiceRecomputeState]:
    """  Administrateurs uniquement. Annule le recalcul en cours : seules ses propres tâches sont annulées,
    les empreintes déjà recalculées restent, le rapport est figé et la bibliothèque rapprochée de
    nouveau (voice_recompute_not_running sans recalcul en cours).

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceRecomputeState]
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

) -> Error | VoiceRecomputeState | None:
    """  Administrateurs uniquement. Annule le recalcul en cours : seules ses propres tâches sont annulées,
    les empreintes déjà recalculées restent, le rapport est figé et la bibliothèque rapprochée de
    nouveau (voice_recompute_not_running sans recalcul en cours).

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceRecomputeState
     """


    return (await asyncio_detailed(
        client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
