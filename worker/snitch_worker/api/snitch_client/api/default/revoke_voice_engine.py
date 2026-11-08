from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.revoke_voice_engine_response_200 import RevokeVoiceEngineResponse200
from typing import cast



def _get_kwargs(
    digest: str,
    *,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/admin/voice-engines/{digest}".format(digest=quote(str(digest), safe=""),),
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | RevokeVoiceEngineResponse200:
    if response.status_code == 200:
        response_200 = RevokeVoiceEngineResponse200.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | RevokeVoiceEngineResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    digest: str,
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | RevokeVoiceEngineResponse200]:
    """  Super-administrateur : révoque une version et annule ses ordres non encore acquittés. Ne retire
    aucun runtime installé.

    Args:
        digest (str):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | RevokeVoiceEngineResponse200]
     """


    kwargs = _get_kwargs(
        digest=digest,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    digest: str,
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> Error | RevokeVoiceEngineResponse200 | None:
    """  Super-administrateur : révoque une version et annule ses ordres non encore acquittés. Ne retire
    aucun runtime installé.

    Args:
        digest (str):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | RevokeVoiceEngineResponse200
     """


    return sync_detailed(
        digest=digest,
client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    digest: str,
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | RevokeVoiceEngineResponse200]:
    """  Super-administrateur : révoque une version et annule ses ordres non encore acquittés. Ne retire
    aucun runtime installé.

    Args:
        digest (str):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | RevokeVoiceEngineResponse200]
     """


    kwargs = _get_kwargs(
        digest=digest,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    digest: str,
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> Error | RevokeVoiceEngineResponse200 | None:
    """  Super-administrateur : révoque une version et annule ses ordres non encore acquittés. Ne retire
    aucun runtime installé.

    Args:
        digest (str):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | RevokeVoiceEngineResponse200
     """


    return (await asyncio_detailed(
        digest=digest,
client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
