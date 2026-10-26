from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.scheduled_voice import ScheduledVoice
from ...models.voice_schedule_input import VoiceScheduleInput
from typing import cast



def _get_kwargs(
    id: int,
    *,
    body: VoiceScheduleInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/clips/{id}/voice".format(id=quote(str(id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | ScheduledVoice:
    if response.status_code == 200:
        response_200 = ScheduledVoice.from_dict(response.json())



        return response_200

    if response.status_code == 201:
        response_201 = ScheduledVoice.from_dict(response.json())



        return response_201

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | ScheduledVoice]:
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
    body: VoiceScheduleInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | ScheduledVoice]:
    """  Administrateurs uniquement. Action explicite pour un clip déjà transcrit ; la transcription n'est
    jamais relancée. Les empreintes sont planifiées automatiquement à l'achèvement de la diarisation.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (VoiceScheduleInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ScheduledVoice]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    *,
    client: AuthenticatedClient,
    body: VoiceScheduleInput,
    origin: str,
    x_csrf_token: str,

) -> Error | ScheduledVoice | None:
    """  Administrateurs uniquement. Action explicite pour un clip déjà transcrit ; la transcription n'est
    jamais relancée. Les empreintes sont planifiées automatiquement à l'achèvement de la diarisation.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (VoiceScheduleInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ScheduledVoice
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    body: VoiceScheduleInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | ScheduledVoice]:
    """  Administrateurs uniquement. Action explicite pour un clip déjà transcrit ; la transcription n'est
    jamais relancée. Les empreintes sont planifiées automatiquement à l'achèvement de la diarisation.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (VoiceScheduleInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ScheduledVoice]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    *,
    client: AuthenticatedClient,
    body: VoiceScheduleInput,
    origin: str,
    x_csrf_token: str,

) -> Error | ScheduledVoice | None:
    """  Administrateurs uniquement. Action explicite pour un clip déjà transcrit ; la transcription n'est
    jamais relancée. Les empreintes sont planifiées automatiquement à l'achèvement de la diarisation.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (VoiceScheduleInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ScheduledVoice
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
