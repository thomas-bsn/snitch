from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.audio_track_override import AudioTrackOverride
from ...models.corrections_warning import CorrectionsWarning
from ...models.error import Error
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    id: int,
    *,
    body: AudioTrackOverride,
    confirm_replace_corrections: bool | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token





    params: dict[str, Any] = {}

    params["confirm_replace_corrections"] = confirm_replace_corrections


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/admin/clips/{id}/audio-track".format(id=quote(str(id), safe=""),),
        "params": params,
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AudioTrackOverride | CorrectionsWarning | Error:
    if response.status_code == 200:
        response_200 = AudioTrackOverride.from_dict(response.json())



        return response_200

    if response.status_code == 409:
        response_409 = CorrectionsWarning.from_dict(response.json())



        return response_409

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AudioTrackOverride | CorrectionsWarning | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    body: AudioTrackOverride,
    confirm_replace_corrections: bool | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[AudioTrackOverride | CorrectionsWarning | Error]:
    """
    Args:
        id (int):
        confirm_replace_corrections (bool | Unset):
        origin (str):
        x_csrf_token (str):
        body (AudioTrackOverride):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AudioTrackOverride | CorrectionsWarning | Error]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
confirm_replace_corrections=confirm_replace_corrections,
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
    client: AuthenticatedClient | Client,
    body: AudioTrackOverride,
    confirm_replace_corrections: bool | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> AudioTrackOverride | CorrectionsWarning | Error | None:
    """
    Args:
        id (int):
        confirm_replace_corrections (bool | Unset):
        origin (str):
        x_csrf_token (str):
        body (AudioTrackOverride):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AudioTrackOverride | CorrectionsWarning | Error
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
confirm_replace_corrections=confirm_replace_corrections,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    body: AudioTrackOverride,
    confirm_replace_corrections: bool | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[AudioTrackOverride | CorrectionsWarning | Error]:
    """
    Args:
        id (int):
        confirm_replace_corrections (bool | Unset):
        origin (str):
        x_csrf_token (str):
        body (AudioTrackOverride):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AudioTrackOverride | CorrectionsWarning | Error]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
confirm_replace_corrections=confirm_replace_corrections,
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
    client: AuthenticatedClient | Client,
    body: AudioTrackOverride,
    confirm_replace_corrections: bool | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> AudioTrackOverride | CorrectionsWarning | Error | None:
    """
    Args:
        id (int):
        confirm_replace_corrections (bool | Unset):
        origin (str):
        x_csrf_token (str):
        body (AudioTrackOverride):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AudioTrackOverride | CorrectionsWarning | Error
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
confirm_replace_corrections=confirm_replace_corrections,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
