from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.discord_bot_input import DiscordBotInput
from ...models.discord_bot_status import DiscordBotStatus
from ...models.error import Error
from typing import cast



def _get_kwargs(
    *,
    body: DiscordBotInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/admin/discord-bot",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> DiscordBotStatus | Error:
    if response.status_code == 200:
        response_200 = DiscordBotStatus.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[DiscordBotStatus | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: DiscordBotInput,
    origin: str,
    x_csrf_token: str,

) -> Response[DiscordBotStatus | Error]:
    """  Réservé au Super-administrateur : vérifie le token auprès de Discord puis l'enregistre chiffré. Un
    token refusé n'est pas enregistré (invalid_discord_bot_token) et n'est jamais renvoyé.

    Args:
        origin (str):
        x_csrf_token (str):
        body (DiscordBotInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DiscordBotStatus | Error]
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
    client: AuthenticatedClient | Client,
    body: DiscordBotInput,
    origin: str,
    x_csrf_token: str,

) -> DiscordBotStatus | Error | None:
    """  Réservé au Super-administrateur : vérifie le token auprès de Discord puis l'enregistre chiffré. Un
    token refusé n'est pas enregistré (invalid_discord_bot_token) et n'est jamais renvoyé.

    Args:
        origin (str):
        x_csrf_token (str):
        body (DiscordBotInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DiscordBotStatus | Error
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: DiscordBotInput,
    origin: str,
    x_csrf_token: str,

) -> Response[DiscordBotStatus | Error]:
    """  Réservé au Super-administrateur : vérifie le token auprès de Discord puis l'enregistre chiffré. Un
    token refusé n'est pas enregistré (invalid_discord_bot_token) et n'est jamais renvoyé.

    Args:
        origin (str):
        x_csrf_token (str):
        body (DiscordBotInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DiscordBotStatus | Error]
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
    client: AuthenticatedClient | Client,
    body: DiscordBotInput,
    origin: str,
    x_csrf_token: str,

) -> DiscordBotStatus | Error | None:
    """  Réservé au Super-administrateur : vérifie le token auprès de Discord puis l'enregistre chiffré. Un
    token refusé n'est pas enregistré (invalid_discord_bot_token) et n'est jamais renvoyé.

    Args:
        origin (str):
        x_csrf_token (str):
        body (DiscordBotInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DiscordBotStatus | Error
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
