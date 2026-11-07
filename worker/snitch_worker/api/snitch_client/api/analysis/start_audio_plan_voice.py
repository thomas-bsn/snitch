from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.audio_plan_voice import AudioPlanVoice
from ...models.audio_plan_voice_confirmation import AudioPlanVoiceConfirmation
from ...models.audio_plan_voice_start import AudioPlanVoiceStart
from typing import cast



def _get_kwargs(
    id: int,
    plan_id: int,
    *,
    body: AudioPlanVoiceStart,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/clips/{id}/audio-plans/{plan_id}/voice".format(id=quote(str(id), safe=""),plan_id=quote(str(plan_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | AudioPlanVoice | AudioPlanVoiceConfirmation | None:
    if response.status_code == 200:
        response_200 = AudioPlanVoice.from_dict(response.json())



        return response_200

    if response.status_code == 400:
        response_400 = cast(Any, None)
        return response_400

    if response.status_code == 401:
        response_401 = cast(Any, None)
        return response_401

    if response.status_code == 403:
        response_403 = cast(Any, None)
        return response_403

    if response.status_code == 404:
        response_404 = cast(Any, None)
        return response_404

    if response.status_code == 409:
        response_409 = AudioPlanVoiceConfirmation.from_dict(response.json())



        return response_409

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | AudioPlanVoice | AudioPlanVoiceConfirmation]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    plan_id: int,
    *,
    client: AuthenticatedClient | Client,
    body: AudioPlanVoiceStart,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | AudioPlanVoice | AudioPlanVoiceConfirmation]:
    """
    Args:
        id (int):
        plan_id (int):
        origin (str):
        x_csrf_token (str):
        body (AudioPlanVoiceStart):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | AudioPlanVoice | AudioPlanVoiceConfirmation]
     """


    kwargs = _get_kwargs(
        id=id,
plan_id=plan_id,
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
    plan_id: int,
    *,
    client: AuthenticatedClient | Client,
    body: AudioPlanVoiceStart,
    origin: str,
    x_csrf_token: str,

) -> Any | AudioPlanVoice | AudioPlanVoiceConfirmation | None:
    """
    Args:
        id (int):
        plan_id (int):
        origin (str):
        x_csrf_token (str):
        body (AudioPlanVoiceStart):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | AudioPlanVoice | AudioPlanVoiceConfirmation
     """


    return sync_detailed(
        id=id,
plan_id=plan_id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    plan_id: int,
    *,
    client: AuthenticatedClient | Client,
    body: AudioPlanVoiceStart,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | AudioPlanVoice | AudioPlanVoiceConfirmation]:
    """
    Args:
        id (int):
        plan_id (int):
        origin (str):
        x_csrf_token (str):
        body (AudioPlanVoiceStart):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | AudioPlanVoice | AudioPlanVoiceConfirmation]
     """


    kwargs = _get_kwargs(
        id=id,
plan_id=plan_id,
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
    plan_id: int,
    *,
    client: AuthenticatedClient | Client,
    body: AudioPlanVoiceStart,
    origin: str,
    x_csrf_token: str,

) -> Any | AudioPlanVoice | AudioPlanVoiceConfirmation | None:
    """
    Args:
        id (int):
        plan_id (int):
        origin (str):
        x_csrf_token (str):
        body (AudioPlanVoiceStart):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | AudioPlanVoice | AudioPlanVoiceConfirmation
     """


    return (await asyncio_detailed(
        id=id,
plan_id=plan_id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
