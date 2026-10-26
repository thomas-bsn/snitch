from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.account_preferences import AccountPreferences
from ...models.account_preferences_input import AccountPreferencesInput
from ...models.error import Error
from typing import cast



def _get_kwargs(
    *,
    body: AccountPreferencesInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/account/preferences",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AccountPreferences | Error:
    if response.status_code == 200:
        response_200 = AccountPreferences.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AccountPreferences | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: AccountPreferencesInput,
    origin: str,
    x_csrf_token: str,

) -> Response[AccountPreferences | Error]:
    """  Enregistre la langue d'interface du compte connecté (en ou fr). Toute autre valeur est refusée avec
    400 invalid_language ; propriétés supplémentaires ou absence de langue : 400 invalid_request. Exige
    session active, même origine et X-CSRF-Token.

    Args:
        origin (str):
        x_csrf_token (str):
        body (AccountPreferencesInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AccountPreferences | Error]
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
    body: AccountPreferencesInput,
    origin: str,
    x_csrf_token: str,

) -> AccountPreferences | Error | None:
    """  Enregistre la langue d'interface du compte connecté (en ou fr). Toute autre valeur est refusée avec
    400 invalid_language ; propriétés supplémentaires ou absence de langue : 400 invalid_request. Exige
    session active, même origine et X-CSRF-Token.

    Args:
        origin (str):
        x_csrf_token (str):
        body (AccountPreferencesInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AccountPreferences | Error
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
    body: AccountPreferencesInput,
    origin: str,
    x_csrf_token: str,

) -> Response[AccountPreferences | Error]:
    """  Enregistre la langue d'interface du compte connecté (en ou fr). Toute autre valeur est refusée avec
    400 invalid_language ; propriétés supplémentaires ou absence de langue : 400 invalid_request. Exige
    session active, même origine et X-CSRF-Token.

    Args:
        origin (str):
        x_csrf_token (str):
        body (AccountPreferencesInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AccountPreferences | Error]
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
    body: AccountPreferencesInput,
    origin: str,
    x_csrf_token: str,

) -> AccountPreferences | Error | None:
    """  Enregistre la langue d'interface du compte connecté (en ou fr). Toute autre valeur est refusée avec
    400 invalid_language ; propriétés supplémentaires ou absence de langue : 400 invalid_request. Exige
    session active, même origine et X-CSRF-Token.

    Args:
        origin (str):
        x_csrf_token (str):
        body (AccountPreferencesInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AccountPreferences | Error
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
