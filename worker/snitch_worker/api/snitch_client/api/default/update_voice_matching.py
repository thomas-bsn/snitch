from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_matching_input import VoiceMatchingInput
from ...models.voice_matching_settings import VoiceMatchingSettings
from typing import cast



def _get_kwargs(
    *,
    body: VoiceMatchingInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/admin/voice-matching",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceMatchingSettings:
    if response.status_code == 200:
        response_200 = VoiceMatchingSettings.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceMatchingSettings]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: VoiceMatchingInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceMatchingSettings]:
    """  Administrateurs uniquement. L'interrupteur d'automatisation n'est pas modifiable ici (décision
    d'activation séparée). Un changement de seuil réexamine les empreintes existantes, sans worker ni
    vidéo.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceMatchingInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceMatchingSettings]
     """


    kwargs = _get_kwargs(
        body=body,
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
    body: VoiceMatchingInput,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceMatchingSettings | None:
    """  Administrateurs uniquement. L'interrupteur d'automatisation n'est pas modifiable ici (décision
    d'activation séparée). Un changement de seuil réexamine les empreintes existantes, sans worker ni
    vidéo.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceMatchingInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceMatchingSettings
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: VoiceMatchingInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceMatchingSettings]:
    """  Administrateurs uniquement. L'interrupteur d'automatisation n'est pas modifiable ici (décision
    d'activation séparée). Un changement de seuil réexamine les empreintes existantes, sans worker ni
    vidéo.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceMatchingInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceMatchingSettings]
     """


    kwargs = _get_kwargs(
        body=body,
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
    body: VoiceMatchingInput,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceMatchingSettings | None:
    """  Administrateurs uniquement. L'interrupteur d'automatisation n'est pas modifiable ici (décision
    d'activation séparée). Un changement de seuil réexamine les empreintes existantes, sans worker ni
    vidéo.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceMatchingInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceMatchingSettings
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
