from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.transcription_settings_input import TranscriptionSettingsInput
from ...models.transcription_settings_state import TranscriptionSettingsState
from typing import cast



def _get_kwargs(
    *,
    body: TranscriptionSettingsInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/admin/transcription-settings",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | TranscriptionSettingsState | None:
    if response.status_code == 200:
        response_200 = TranscriptionSettingsState.from_dict(response.json())



        return response_200

    if response.status_code == 400:
        response_400 = cast(Any, None)
        return response_400

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | TranscriptionSettingsState]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: TranscriptionSettingsInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | TranscriptionSettingsState]:
    """
    Args:
        origin (str):
        x_csrf_token (str):
        body (TranscriptionSettingsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | TranscriptionSettingsState]
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
    body: TranscriptionSettingsInput,
    origin: str,
    x_csrf_token: str,

) -> Any | TranscriptionSettingsState | None:
    """
    Args:
        origin (str):
        x_csrf_token (str):
        body (TranscriptionSettingsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | TranscriptionSettingsState
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
    body: TranscriptionSettingsInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | TranscriptionSettingsState]:
    """
    Args:
        origin (str):
        x_csrf_token (str):
        body (TranscriptionSettingsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | TranscriptionSettingsState]
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
    body: TranscriptionSettingsInput,
    origin: str,
    x_csrf_token: str,

) -> Any | TranscriptionSettingsState | None:
    """
    Args:
        origin (str):
        x_csrf_token (str):
        body (TranscriptionSettingsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | TranscriptionSettingsState
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
