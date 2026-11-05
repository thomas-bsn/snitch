from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.avatar_settings import AvatarSettings
from ...models.error import Error
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
        "method": "post",
        "url": "/api/account/avatar/discord-refresh",
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AvatarSettings | Error:
    if response.status_code == 200:
        response_200 = AvatarSettings.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AvatarSettings | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> Response[AvatarSettings | Error]:
    """  Actualise la photo Discord du compte connecté. Ne touche jamais la photo personnelle. Une fois
    toutes les 30 secondes au plus par compte.

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AvatarSettings | Error]
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
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> AvatarSettings | Error | None:
    """  Actualise la photo Discord du compte connecté. Ne touche jamais la photo personnelle. Une fois
    toutes les 30 secondes au plus par compte.

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AvatarSettings | Error
     """


    return sync_detailed(
        client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> Response[AvatarSettings | Error]:
    """  Actualise la photo Discord du compte connecté. Ne touche jamais la photo personnelle. Une fois
    toutes les 30 secondes au plus par compte.

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AvatarSettings | Error]
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
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> AvatarSettings | Error | None:
    """  Actualise la photo Discord du compte connecté. Ne touche jamais la photo personnelle. Une fois
    toutes les 30 secondes au plus par compte.

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AvatarSettings | Error
     """


    return (await asyncio_detailed(
        client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
