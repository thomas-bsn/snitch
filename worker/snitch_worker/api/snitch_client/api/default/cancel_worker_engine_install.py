from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.engine_recovery_result import EngineRecoveryResult
from ...models.error import Error
from typing import cast



def _get_kwargs(
    id: str,
    digest: str,
    *,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/workers/{id}/voice-engines/{digest}/cancel".format(id=quote(str(id), safe=""),digest=quote(str(digest), safe=""),),
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> EngineRecoveryResult | Error:
    if response.status_code == 200:
        response_200 = EngineRecoveryResult.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[EngineRecoveryResult | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: str,
    digest: str,
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> Response[EngineRecoveryResult | Error]:
    """  Administrateur : annule une installation. Un ordre non acquitté est annulé aussitôt (cancelled) ; un
    ordre en cours est marqué, livré au prochain relevé et confirmé par le Worker à son prochain point
    sûr (cancelling). Répéter la demande est sans effet.

    Args:
        id (str):
        digest (str):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EngineRecoveryResult | Error]
     """


    kwargs = _get_kwargs(
        id=id,
digest=digest,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: str,
    digest: str,
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> EngineRecoveryResult | Error | None:
    """  Administrateur : annule une installation. Un ordre non acquitté est annulé aussitôt (cancelled) ; un
    ordre en cours est marqué, livré au prochain relevé et confirmé par le Worker à son prochain point
    sûr (cancelling). Répéter la demande est sans effet.

    Args:
        id (str):
        digest (str):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EngineRecoveryResult | Error
     """


    return sync_detailed(
        id=id,
digest=digest,
client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: str,
    digest: str,
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> Response[EngineRecoveryResult | Error]:
    """  Administrateur : annule une installation. Un ordre non acquitté est annulé aussitôt (cancelled) ; un
    ordre en cours est marqué, livré au prochain relevé et confirmé par le Worker à son prochain point
    sûr (cancelling). Répéter la demande est sans effet.

    Args:
        id (str):
        digest (str):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EngineRecoveryResult | Error]
     """


    kwargs = _get_kwargs(
        id=id,
digest=digest,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: str,
    digest: str,
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> EngineRecoveryResult | Error | None:
    """  Administrateur : annule une installation. Un ordre non acquitté est annulé aussitôt (cancelled) ; un
    ordre en cours est marqué, livré au prochain relevé et confirmé par le Worker à son prochain point
    sûr (cancelling). Répéter la demande est sans effet.

    Args:
        id (str):
        digest (str):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EngineRecoveryResult | Error
     """


    return (await asyncio_detailed(
        id=id,
digest=digest,
client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
